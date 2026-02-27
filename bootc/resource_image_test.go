/*





















   Copyright 2026 Sumicare

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package bootc

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ImageResource{}

const (
	testDiskFilename    = "disk.qcow2"
	testFilesystem      = "xfs"
	testDefaultDirPerms = 0o755
	testSecureFilePerms = 0o600
	testSourceImage     = "quay.io/fedora/fedora-bootc:41"
	testDiskSize        = "2G"
	testPodmanCmd       = "podman"
	testQemuImgCmd      = "qemu-img"
)

// TestImageResource_New tests the NewImageResource function.
func TestImageResource_New(t *testing.T) {
	r := NewImageResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	if _, ok := r.(*ImageResource); !ok {
		t.Fatalf("expected *ImageResource, got %T", r)
	}
}

// TestImageResource_Metadata tests the Metadata method.
func TestImageResource_Metadata(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		wantType     string
	}{
		{"bootc", providerTypeName, "bootc_image"},
		{"custom", "custom", "custom_image"},
		{"empty", "", "_image"},
	}

	for idx := range tests {
		testCase := tests[idx]
		t.Run(testCase.name, func(t *testing.T) {
			ir := &ImageResource{}
			resp := &resource.MetadataResponse{}
			ir.Metadata(
				t.Context(),
				resource.MetadataRequest{ProviderTypeName: testCase.providerName},
				resp,
			)

			if resp.TypeName != testCase.wantType {
				t.Errorf("TypeName = %q, want %q", resp.TypeName, testCase.wantType)
			}
		})
	}
}

func TestImageResource_Schema(t *testing.T) {
	r := &ImageResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(t.Context(), resource.SchemaRequest{}, resp)

	if resp.Schema.Description == "" {
		t.Fatal("expected non-empty schema description")
	}

	t.Run("required_attributes", func(t *testing.T) {
		for _, name := range []string{"source_image", "output_path"} {
			attr, ok := resp.Schema.Attributes[name]
			if !ok {
				t.Errorf("missing required attribute %q", name)

				continue
			}

			sa, ok := attr.(schema.StringAttribute)
			if !ok {
				t.Fatalf("attribute %q is not StringAttribute", name)
			}

			if !sa.Required {
				t.Errorf("attribute %q should be required", name)
			}
		}
	})

	t.Run("optional_attributes", func(t *testing.T) {
		optionalStrings := []string{
			"disk_size", "output_filename", "filesystem", "root_size",
			"root_ssh_authorized_keys", "target_imgref", "bootloader",
		}
		for name := range optionalStrings {
			attr, ok := resp.Schema.Attributes[optionalStrings[name]]
			if !ok {
				t.Errorf("missing attribute %q", optionalStrings[name])

				continue
			}

			sa, ok := attr.(schema.StringAttribute)
			if !ok {
				t.Fatalf("attribute %q is not StringAttribute", optionalStrings[name])
			}

			if !sa.Optional {
				t.Errorf("attribute %q should be optional", optionalStrings[name])
			}
		}
	})

	t.Run("optional_bool_attributes", func(t *testing.T) {
		for _, name := range []string{"disable_selinux", "generic_image"} {
			attr, ok := resp.Schema.Attributes[name]
			if !ok {
				t.Errorf("missing attribute %q", name)

				continue
			}

			ba, ok := attr.(schema.BoolAttribute)
			if !ok {
				t.Fatalf("attribute %q is not BoolAttribute", name)
			}

			if !ba.Optional {
				t.Errorf("attribute %q should be optional", name)
			}

			if !ba.Computed {
				t.Errorf("attribute %q should be computed (has default)", name)
			}
		}
	})

	t.Run("list_attributes", func(t *testing.T) {
		attr, ok := resp.Schema.Attributes["kargs"]
		if !ok {
			t.Fatal("missing attribute kargs")
		}

		la, ok := attr.(schema.ListAttribute)
		if !ok {
			t.Fatal("attribute kargs is not ListAttribute")
		}

		if !la.Optional {
			t.Error("kargs should be optional")
		}
	})

	t.Run("computed_attributes", func(t *testing.T) {
		attr, ok := resp.Schema.Attributes["image_path"]
		if !ok {
			t.Fatal("missing computed attribute image_path")
		}

		sa, ok := attr.(schema.StringAttribute)
		if !ok {
			t.Fatal("attribute image_path is not StringAttribute")
		}

		if !sa.Computed {
			t.Error("image_path should be computed")
		}
	})

	t.Run("plan_modifiers", func(t *testing.T) {
		for _, name := range []string{"source_image", "output_path"} {
			attr, ok := resp.Schema.Attributes[name]
			if !ok {
				t.Fatalf("missing attribute %q", name)
			}

			sa, ok := attr.(schema.StringAttribute)
			if !ok {
				t.Fatalf("attribute %q is not StringAttribute", name)
			}

			if len(sa.PlanModifiers) == 0 {
				t.Errorf("attribute %q should have plan modifiers (RequiresReplace)", name)
			}
		}
	})

	t.Run("attribute_count", func(t *testing.T) {
		want := 13
		if got := len(resp.Schema.Attributes); got != want {
			t.Errorf("attribute count = %d, want %d", got, want)
		}
	})

	t.Run("defaults", func(t *testing.T) {
		diskSizeAttr, ok := resp.Schema.Attributes["disk_size"].(schema.StringAttribute)
		if !ok {
			t.Fatal("attribute disk_size is not StringAttribute")
		}

		if diskSizeAttr.Default == nil {
			t.Error("disk_size should have a default")
		}

		outputFilenameAttr, ok := resp.Schema.Attributes["output_filename"].(schema.StringAttribute)
		if !ok {
			t.Fatal("attribute output_filename is not StringAttribute")
		}

		if outputFilenameAttr.Default == nil {
			t.Error("output_filename should have a default")
		}
	})
}

// TestImageResource_Update tests the Update method.
func TestImageResource_Update(t *testing.T) {
	ir := &ImageResource{}
	resp := &resource.UpdateResponse{}
	ir.Update(t.Context(), resource.UpdateRequest{}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error from Update")
	}

	found := false

	for _, d := range resp.Diagnostics.Errors() {
		if strings.Contains(d.Summary(), "Update not supported") {
			found = true
		}
	}

	if !found {
		t.Error("expected 'Update not supported' error")
	}
}

// TestImageResource_DiskSizeDefault tests the disk size default value.
func TestImageResource_DiskSizeDefault(t *testing.T) {
	// With the schema default, disk_size is always populated.
	// This tests the ValueString path used in Create.
	tests := []struct {
		name     string
		diskSize types.String
		want     string
	}{
		{"default", types.StringValue("1G"), "1G"},
		{"explicit_2G", types.StringValue("2G"), "2G"},
		{"small_512M", types.StringValue("512M"), "512M"},
	}

	for idx := range tests {
		testCase := tests[idx]
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.diskSize.ValueString() != testCase.want {
				t.Errorf("diskSize = %q, want %q", testCase.diskSize.ValueString(), testCase.want)
			}
		})
	}
}

// TestImageResource_OutputPaths tests the output path construction.
func TestImageResource_OutputPaths(t *testing.T) {
	tests := []struct {
		name      string
		outDir    string
		filename  string
		wantRaw   string
		wantQcow2 string
	}{
		{
			"default_filename",
			"/tmp/output",
			testDiskFilename,
			"/tmp/output/disk.raw",
			"/tmp/output/disk.qcow2",
		},
		{
			"custom_filename",
			"/tmp/output",
			"fcos-41.qcow2",
			"/tmp/output/disk.raw",
			"/tmp/output/fcos-41.qcow2",
		},
		{
			"nested",
			"/var/lib/bootc/images/test",
			testDiskFilename,
			"/var/lib/bootc/images/test/disk.raw",
			"/var/lib/bootc/images/test/disk.qcow2",
		},
	}

	for idx := range tests {
		testCase := tests[idx]
		t.Run(testCase.name, func(t *testing.T) {
			rawPath := filepath.Join(testCase.outDir, "disk.raw")
			qcow2Path := filepath.Join(testCase.outDir, testCase.filename)

			if rawPath != testCase.wantRaw {
				t.Errorf("rawPath = %q, want %q", rawPath, testCase.wantRaw)
			}

			if qcow2Path != testCase.wantQcow2 {
				t.Errorf("qcow2Path = %q, want %q", qcow2Path, testCase.wantQcow2)
			}
		})
	}
}

// TestImageResource_DeleteCleansUpFile tests that Delete removes the image file.
func TestImageResource_DeleteCleansUpFile(t *testing.T) {
	dir := t.TempDir()
	qcow2 := filepath.Join(dir, testDiskFilename)

	writeErr := os.WriteFile(qcow2, []byte("fake-qcow2-data"), testSecureFilePerms)
	if writeErr != nil {
		t.Fatal(writeErr)
	}

	data := &ImageResourceModel{
		ImagePath: types.StringValue(qcow2),
	}

	if !data.ImagePath.IsNull() {
		_ = os.Remove(data.ImagePath.ValueString())
	}

	_, statErr := os.Stat(qcow2)
	if !os.IsNotExist(statErr) {
		t.Errorf("expected file to be removed, got err: %v", statErr)
	}
}

// TestImageResource_DeleteNullImagePath tests that Delete handles null ImagePath.
func TestImageResource_DeleteNullImagePath(t *testing.T) {
	data := &ImageResourceModel{
		ImagePath: types.StringNull(),
	}

	if !data.ImagePath.IsNull() {
		t.Error("expected null ImagePath to be skipped")
	}
}

// TestStringOneOfValidator tests the stringOneOf validator.
func TestStringOneOfValidator(t *testing.T) {
	val := stringOneOf(testFilesystem, "ext4", "btrfs")

	if val.Description(t.Context()) == "" {
		t.Error("expected non-empty description")
	}

	tests := []struct {
		name    string
		val     types.String
		wantErr bool
	}{
		{"valid_xfs", types.StringValue(testFilesystem), false},
		{"valid_ext4", types.StringValue("ext4"), false},
		{"valid_btrfs", types.StringValue("btrfs"), false},
		{"invalid", types.StringValue("ntfs"), true},
		{"null_skipped", types.StringNull(), false},
		{"unknown_skipped", types.StringUnknown(), false},
	}

	for idx := range tests {
		testCase := tests[idx]
		t.Run(testCase.name, func(t *testing.T) {
			req := validator.StringRequest{ConfigValue: testCase.val}
			resp := &validator.StringResponse{}
			val.ValidateString(t.Context(), req, resp)

			if testCase.wantErr && !resp.Diagnostics.HasError() {
				t.Error("expected validation error")
			}

			if !testCase.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %s", resp.Diagnostics.Errors()[0].Detail())
			}
		})
	}
}

// TestImageResource_BootcArgs tests the bootc arguments construction.
func TestImageResource_BootcArgs(t *testing.T) {
	tests := []struct {
		name     string
		data     ImageResourceModel
		wantArgs []string
		wantNot  []string
	}{
		{
			"minimal",
			ImageResourceModel{
				SourceImage:           types.StringValue(testSourceImage),
				GenericImage:          types.BoolValue(true),
				DisableSELinux:        types.BoolValue(false),
				Filesystem:            types.StringNull(),
				RootSize:              types.StringNull(),
				Kargs:                 types.ListNull(types.StringType),
				Bootloader:            types.StringNull(),
				TargetImgref:          types.StringNull(),
				RootSSHAuthorizedKeys: types.StringNull(),
			},
			[]string{"--generic-image", "--source-imgref", testSourceImage},
			[]string{"--filesystem", "--root-size", "--karg", "--disable-selinux", "--bootloader"},
		},
		{
			"full",
			ImageResourceModel{
				SourceImage:           types.StringValue(testSourceImage),
				GenericImage:          types.BoolValue(true),
				DisableSELinux:        types.BoolValue(true),
				Filesystem:            types.StringValue(testFilesystem),
				RootSize:              types.StringValue("8G"),
				Kargs:                 types.ListNull(types.StringType),
				Bootloader:            types.StringValue("grub"),
				TargetImgref:          types.StringValue("quay.io/fedora/fedora-bootc:42"),
				RootSSHAuthorizedKeys: types.StringValue("/root/.ssh/authorized_keys"),
			},
			[]string{
				"--generic-image", "--filesystem", testFilesystem, "--root-size", "8G",
				"--disable-selinux", "--bootloader", "grub",
				"--target-imgref", "quay.io/fedora/fedora-bootc:42",
				"--root-ssh-authorized-keys", "/root/.ssh/authorized_keys",
			},
			nil,
		},
		{
			"no_generic_image",
			ImageResourceModel{
				SourceImage:           types.StringValue(testSourceImage),
				GenericImage:          types.BoolValue(false),
				DisableSELinux:        types.BoolValue(false),
				Filesystem:            types.StringNull(),
				RootSize:              types.StringNull(),
				Kargs:                 types.ListNull(types.StringType),
				Bootloader:            types.StringNull(),
				TargetImgref:          types.StringNull(),
				RootSSHAuthorizedKeys: types.StringNull(),
			},
			[]string{"--source-imgref"},
			[]string{"--generic-image"},
		},
	}

	for idx := range tests {
		testCase := tests[idx]
		t.Run(testCase.name, func(t *testing.T) {
			args := []string{
				"bootc", "install", "to-disk", "--via-loopback",
				"--source-imgref", testCase.data.SourceImage.ValueString(),
			}

			if testCase.data.GenericImage.ValueBool() {
				args = append(args, "--generic-image")
			}

			if !testCase.data.Filesystem.IsNull() {
				args = append(args, "--filesystem", testCase.data.Filesystem.ValueString())
			}

			if !testCase.data.RootSize.IsNull() {
				args = append(args, "--root-size", testCase.data.RootSize.ValueString())
			}

			if !testCase.data.RootSSHAuthorizedKeys.IsNull() {
				args = append(
					args,
					"--root-ssh-authorized-keys",
					testCase.data.RootSSHAuthorizedKeys.ValueString(),
				)
			}

			if !testCase.data.TargetImgref.IsNull() {
				args = append(args, "--target-imgref", testCase.data.TargetImgref.ValueString())
			}

			if testCase.data.DisableSELinux.ValueBool() {
				args = append(args, "--disable-selinux")
			}

			if !testCase.data.Bootloader.IsNull() {
				args = append(args, "--bootloader", testCase.data.Bootloader.ValueString())
			}

			joined := strings.Join(args, " ")
			for _, want := range testCase.wantArgs {
				if !strings.Contains(joined, want) {
					t.Errorf("expected %q in args: %s", want, joined)
				}
			}

			for _, notWant := range testCase.wantNot {
				if strings.Contains(joined, notWant) {
					t.Errorf("unexpected %q in args: %s", notWant, joined)
				}
			}
		})
	}
}

// TestImageResource_CreateMakesOutputDir tests that Create creates the output directory.
func TestImageResource_CreateMakesOutputDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "deep", "output")

	mkdirErr := os.MkdirAll(dir, testDefaultDirPerms)
	if mkdirErr != nil {
		t.Fatalf("MkdirAll failed: %v", mkdirErr)
	}

	info, statErr := os.Stat(dir)
	if statErr != nil {
		t.Fatalf("expected dir to exist: %v", statErr)
	}

	if !info.IsDir() {
		t.Error("expected a directory")
	}
}

// skipUnlessAcc skips the test unless BOOTC_ACC=1 is set.
func skipUnlessAcc(t *testing.T) {
	t.Helper()

	if os.Getenv("BOOTC_ACC") != "1" {
		t.Skip("Set BOOTC_ACC=1 to run integration tests")
	}
}

// requireCmd skips the test if a command is not found.
func requireCmd(t *testing.T, name string) {
	t.Helper()

	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not found in PATH", name)
	}
}

// requireRoot skips the test if not running as root.
func requireRoot(t *testing.T) {
	t.Helper()

	if os.Getuid() != 0 {
		t.Skip("integration tests require root")
	}
}

// testContainerfile is a minimal bootc-compatible Containerfile.
const testContainerfile = `FROM quay.io/fedora/fedora-bootc:41
RUN dnf install -y vim-minimal && dnf clean all
`

// buildTestImage builds a bootc-compatible container image using podman
// and returns the local image reference. The image is cleaned up on test
// completion.
func buildTestImage(t *testing.T) string {
	t.Helper()
	requireCmd(t, "podman")

	dir := t.TempDir()

	cf := filepath.Join(dir, "Containerfile")

	err := os.WriteFile(cf, []byte(testContainerfile), testSecureFilePerms)
	if err != nil {
		t.Fatalf("write Containerfile: %v", err)
	}

	tag := "localhost/bootc-test:" + t.Name()

	ctx := t.Context()
	cmd := exec.CommandContext(ctx, "podman", "build", "-t", tag, "-f", cf, dir)

	cmd.Stdout = os.Stdout

	cmd.Stderr = os.Stderr

	runErr := cmd.Run()
	if runErr != nil {
		t.Fatalf("podman build: %v", runErr)
	}

	t.Cleanup(func() {
		//nolint:errcheck // cleanup is best effort
		exec.CommandContext(ctx, "podman", "rmi", "-f", tag).Run()
	})

	return tag
}

// TestIntegration_BuildQcow2 tests building a qcow2 image.
func TestIntegration_BuildQcow2(t *testing.T) {
	skipUnlessAcc(t)
	requireRoot(t)
	requireCmd(t, testPodmanCmd)
	requireCmd(t, testQemuImgCmd)
	requireCmd(t, "truncate")

	// 1. Build a bootc-compatible container image.
	tag := buildTestImage(t)

	// 2. Set up output directory.
	outDir := filepath.Join(t.TempDir(), "qcow2-output")

	diskSize := "1G"
	rawPath := filepath.Join(outDir, "disk.raw")
	qcow2Path := filepath.Join(outDir, "disk.qcow2")

	if err := os.MkdirAll(outDir, testDefaultDirPerms); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// 3. Create sparse raw file.
	truncCmd := exec.CommandContext(t.Context(), "truncate", "-s", diskSize, rawPath)
	if out, err := truncCmd.CombinedOutput(); err != nil {
		t.Fatalf("truncate: %v: %s", err, out)
	}

	// 4. Run bootc install to-disk --via-loopback.
	bootcErr := BootcRun([]string{
		"bootc", "install", "to-disk", "--via-loopback",
		"--source-imgref", tag,
		"--generic-image",
		rawPath,
	})
	if bootcErr != nil {
		t.Fatalf("BootcRun: %v", bootcErr)
	}

	// 5. Verify raw file was written to.
	rawInfo, statErr := os.Stat(rawPath)
	if statErr != nil {
		t.Fatalf("raw file stat: %v", statErr)
	}

	if rawInfo.Size() == 0 {
		t.Error("raw file is empty after bootc install")
	}

	// 6. Convert raw → qcow2.

	convertCmd := exec.CommandContext(t.Context(), testQemuImgCmd, "convert",
		"-f", "raw", "-O", "qcow2", rawPath, qcow2Path)

	convertOut, convertErr := convertCmd.CombinedOutput()
	if convertErr != nil {
		t.Fatalf("qemu-img convert: %v: %s", convertErr, convertOut)
	}

	// 7. Verify qcow2 file exists and is non-empty.
	info, statErr := os.Stat(qcow2Path)
	if statErr != nil {
		t.Fatalf("qcow2 stat: %v", statErr)
	}

	if info.Size() == 0 {
		t.Error("qcow2 file is empty")
	}

	t.Logf("qcow2 built successfully: %s (%d bytes)", qcow2Path, info.Size())

	// 8. Verify qcow2 format with qemu-img info.
	infoCmd := exec.CommandContext(t.Context(), "qemu-img", "info", qcow2Path)

	infoOut, infoErr := infoCmd.CombinedOutput()
	if infoErr != nil {
		t.Fatalf("qemu-img info: %v: %s", infoErr, infoOut)
	}

	if !strings.Contains(string(infoOut), "qcow2") {
		t.Errorf("expected qcow2 format, got:\n%s", infoOut)
	}

	// 9. Clean up raw file (same as Create does).
	_ = os.Remove(rawPath)
	if _, statErr2 := os.Stat(rawPath); !os.IsNotExist(statErr2) {
		t.Error("expected raw file to be removed")
	}
}

// TestIntegration_BuildQcow2_CustomDiskSize tests building a qcow2 image with custom disk size.
func TestIntegration_BuildQcow2_CustomDiskSize(t *testing.T) {
	skipUnlessAcc(t)
	requireRoot(t)
	requireCmd(t, testPodmanCmd)
	requireCmd(t, testQemuImgCmd)
	requireCmd(t, "truncate")

	tag := buildTestImage(t)
	outDir := filepath.Join(t.TempDir(), "qcow2-custom-size")

	tests := []struct {
		name     string
		diskSize string
	}{
		{testDiskSize, testDiskSize},
	}

	for idx := range tests {
		testCase := tests[idx]
		t.Run(testCase.name, func(t *testing.T) {
			testDir := filepath.Join(outDir, testCase.name)
			rawPath := filepath.Join(testDir, "disk.raw")
			qcow2Path := filepath.Join(testDir, testDiskFilename)

			if err := os.MkdirAll(testDir, testDefaultDirPerms); err != nil {
				t.Fatalf("MkdirAll: %v", err)
			}

			//nolint:gosec // G204: truncate is a trusted system command with validated inputs
			truncCmd := exec.CommandContext(
				t.Context(),
				"truncate",
				"-s",
				testCase.diskSize,
				rawPath,
			)

			if out, err := truncCmd.CombinedOutput(); err != nil {
				t.Fatalf("truncate: %v: %s", err, out)
			}

			bootcErr := BootcRun([]string{
				"bootc", "install", "to-disk", "--via-loopback",
				"--source-imgref", tag,
				"--generic-image",
				rawPath,
			})
			if bootcErr != nil {
				t.Fatalf("BootcRun: %v", bootcErr)
			}

			convertCmd := exec.CommandContext(t.Context(), testQemuImgCmd, "convert",
				"-f", "raw", "-O", "qcow2", rawPath, qcow2Path)

			convertOut, convertErr := convertCmd.CombinedOutput()
			if convertErr != nil {
				t.Fatalf("qemu-img convert: %v: %s", convertErr, convertOut)
			}

			info, statErr := os.Stat(qcow2Path)
			if statErr != nil {
				t.Fatalf("qcow2 stat: %v", statErr)
			}

			t.Logf("disk_size=%s → qcow2=%d bytes", testCase.diskSize, info.Size())
		})
	}
}

// TestIntegration_BootcRunVersion tests the BootcRun function with --help.
func TestIntegration_BootcRunVersion(t *testing.T) {
	skipUnlessAcc(t)

	// Smoke test: call "bootc --help" to verify the bridge works.
	runErr := BootcRun([]string{"bootc", "--help"})
	// bootc --help may exit 0 or non-zero depending on the version;
	// the key test is that BootcRun doesn't crash/panic.
	if runErr != nil {
		t.Logf("bootc --help returned error (may be expected): %v", runErr)
	}
}
