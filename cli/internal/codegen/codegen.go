package codegen

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// CompileProto parses a .proto file using protocompile and returns a
// CodeGeneratorRequest that can be piped to any protoc plugin.
func CompileProto(protoDir, protoFile string) (*pluginpb.CodeGeneratorRequest, error) {
	compiler := &protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(
			&protocompile.SourceResolver{
				ImportPaths: []string{protoDir},
			},
		),
		SourceInfoMode: protocompile.SourceInfoStandard,
	}

	linkedFiles, err := compiler.Compile(context.Background(), protoFile)
	if err != nil {
		return nil, fmt.Errorf("compiling proto: %w", err)
	}

	var fileDescriptors []*descriptorpb.FileDescriptorProto
	for _, file := range linkedFiles {
		fileDescriptors = append(fileDescriptors, protodesc.ToFileDescriptorProto(file))
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{protoFile},
		ProtoFile:      fileDescriptors,
	}, nil
}

// RunPlugin invokes a protoc plugin binary with the given CodeGeneratorRequest,
// passing the serialized request on stdin and reading the response from stdout.
func RunPlugin(req *pluginpb.CodeGeneratorRequest, pluginPath string, param string) (*pluginpb.CodeGeneratorResponse, error) {
	r := proto.Clone(req).(*pluginpb.CodeGeneratorRequest)
	if param != "" {
		r.Parameter = proto.String(param)
	}

	data, err := proto.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	cmd := exec.Command(pluginPath)
	cmd.Stdin = bytes.NewReader(data)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running plugin %s: %w\n%s", pluginPath, err, stderr.String())
	}

	var resp pluginpb.CodeGeneratorResponse
	if err := proto.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	if resp.Error != nil && *resp.Error != "" {
		return nil, fmt.Errorf("plugin error: %s", *resp.Error)
	}

	return &resp, nil
}

// RunGoPlugin invokes protoc-gen-go via `go run` so no global install is needed.
func RunGoPlugin(req *pluginpb.CodeGeneratorRequest, param string) (*pluginpb.CodeGeneratorResponse, error) {
	r := proto.Clone(req).(*pluginpb.CodeGeneratorRequest)
	if param != "" {
		r.Parameter = proto.String(param)
	}

	data, err := proto.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	// Try local binary first, fall back to `go run`
	pluginPath, err := exec.LookPath("protoc-gen-go")
	var cmd *exec.Cmd
	if err == nil {
		cmd = exec.Command(pluginPath)
	} else {
		cmd = exec.Command("go", "run", "google.golang.org/protobuf/cmd/protoc-gen-go@latest")
	}

	cmd.Stdin = bytes.NewReader(data)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running protoc-gen-go: %w\n%s", err, stderr.String())
	}

	var resp pluginpb.CodeGeneratorResponse
	if err := proto.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	if resp.Error != nil && *resp.Error != "" {
		return nil, fmt.Errorf("plugin error: %s", *resp.Error)
	}

	return &resp, nil
}

// WriteResponse writes all files from a CodeGeneratorResponse to the output directory.
func WriteResponse(resp *pluginpb.CodeGeneratorResponse, outDir string) ([]string, error) {
	var written []string
	for _, file := range resp.File {
		outPath := filepath.Join(outDir, file.GetName())

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return nil, fmt.Errorf("creating directory for %s: %w", file.GetName(), err)
		}

		if err := os.WriteFile(outPath, []byte(file.GetContent()), 0644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", file.GetName(), err)
		}

		written = append(written, file.GetName())
	}
	return written, nil
}

// HashFile returns the hex-encoded SHA256 hash of a file's contents.
func HashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// ReadStoredHash reads the stored codegen hash from .gapp/codegen.hash.
func ReadStoredHash(projectDir string) string {
	data, err := os.ReadFile(filepath.Join(projectDir, ".gapp", "codegen.hash"))
	if err != nil {
		return ""
	}
	return string(data)
}

// WriteHash writes the codegen hash to .gapp/codegen.hash.
func WriteHash(projectDir, hash string) error {
	dir := filepath.Join(projectDir, ".gapp")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "codegen.hash"), []byte(hash), 0644)
}
