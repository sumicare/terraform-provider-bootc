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
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ImageResource{}

// ImageResource implements the bootc_image Terraform resource.
type ImageResource struct{}

type ImageResourceModel struct {
	Kargs                 types.List   `tfsdk:"kargs"`
	OutputFilename        types.String `tfsdk:"output_filename"`
	DiskSize              types.String `tfsdk:"disk_size"`
	SourceImage           types.String `tfsdk:"source_image"`
	Filesystem            types.String `tfsdk:"filesystem"`
	RootSize              types.String `tfsdk:"root_size"`
	OutputPath            types.String `tfsdk:"output_path"`
	RootSSHAuthorizedKeys types.String `tfsdk:"root_ssh_authorized_keys"`
	TargetImgref          types.String `tfsdk:"target_imgref"`
	Bootloader            types.String `tfsdk:"bootloader"`
	ImagePath             types.String `tfsdk:"image_path"`
	DisableSELinux        types.Bool   `tfsdk:"disable_selinux"`
	GenericImage          types.Bool   `tfsdk:"generic_image"`
}

func NewImageResource() resource.Resource {
	return &ImageResource{}
}

func (*ImageResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_image"
}

func (*ImageResource) Schema(
	_ context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Builds a qcow2 disk image from a bootc container image using bootc install to-disk --via-loopback.",
		Attributes: map[string]schema.Attribute{
			"source_image": schema.StringAttribute{
				Description: "Container image reference (e.g. quay.io/fedora/fedora-coreos:stable).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"output_path": schema.StringAttribute{
				Description: "Directory where the disk image will be written.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"disk_size": schema.StringAttribute{
				Description: "Total raw disk image size (passed to truncate -s). Supports K, M, G, T suffixes.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("1G"),
			},
			"output_filename": schema.StringAttribute{
				Description: "Filename for the resulting qcow2 image within output_path.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("disk.qcow2"),
			},
			"filesystem": schema.StringAttribute{
				Description: "Root filesystem type: xfs, ext4, or btrfs.",
				Optional:    true,
				Validators: []validator.String{
					stringOneOf("xfs", "ext4", "btrfs"),
				},
			},
			"root_size": schema.StringAttribute{
				Description: "Size of the root partition. Allowed suffixes: M (MiB), G (GiB), T (TiB). By default all remaining disk space is used.",
				Optional:    true,
			},
			"kargs": schema.ListAttribute{
				Description: "Kernel arguments to pass to the installed system (e.g. [\"console=ttyS0,115200n8\", \"nosmt\"]).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"root_ssh_authorized_keys": schema.StringAttribute{
				Description: "Path to an authorized_keys file to inject into the root account via systemd tmpfiles.d.",
				Optional:    true,
			},
			"target_imgref": schema.StringAttribute{
				Description: "Container image reference for subsequent bootc upgrades. If unset, defaults to the source image.",
				Optional:    true,
			},
			"disable_selinux": schema.BoolAttribute{
				Description: "Disable SELinux in the installed system.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"generic_image": schema.BoolAttribute{
				Description: "Build a generic disk image (all bootloader types installed, firmware changes skipped). Enabled by default for loopback installs.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"bootloader": schema.StringAttribute{
				Description: "Bootloader to use: grub, systemd, or none.",
				Optional:    true,
				Validators: []validator.String{
					stringOneOf("grub", "systemd", "none"),
				},
			},
			"image_path": schema.StringAttribute{
				Description: "Full path to the resulting qcow2 file.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (*ImageResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data ImageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	outDir := data.OutputPath.ValueString()

	mkdirErr := os.MkdirAll(outDir, 0o755)
	if mkdirErr != nil {
		resp.Diagnostics.AddError("Failed to create output directory", mkdirErr.Error())

		return
	}

	rawPath := filepath.Join(outDir, "disk.raw")
	qcow2Path := filepath.Join(outDir, data.OutputFilename.ValueString())

	// 1. Create sparse raw file
	//nolint:gosec // G204: truncate is a trusted system command with validated inputs
	truncCmd := exec.CommandContext(ctx, "truncate", "-s", data.DiskSize.ValueString(), rawPath)

	truncOut, truncErr := truncCmd.CombinedOutput()
	if truncErr != nil {
		resp.Diagnostics.AddError("Failed to create raw disk image",
			fmt.Sprintf("%v: %s", truncErr, string(truncOut)))

		return
	}

	// 2. Build bootc install args
	args := []string{
		"bootc", "install", "to-disk", "--via-loopback",
		"--source-imgref", data.SourceImage.ValueString(),
	}

	if data.GenericImage.ValueBool() {
		args = append(args, "--generic-image")
	}

	if !data.Filesystem.IsNull() {
		args = append(args, "--filesystem", data.Filesystem.ValueString())
	}

	if !data.RootSize.IsNull() {
		args = append(args, "--root-size", data.RootSize.ValueString())
	}

	if !data.Kargs.IsNull() {
		var kargs []string
		resp.Diagnostics.Append(data.Kargs.ElementsAs(ctx, &kargs, false)...)

		if resp.Diagnostics.HasError() {
			_ = os.Remove(rawPath)

			return
		}

		for _, k := range kargs {
			args = append(args, "--karg", k)
		}
	}

	if !data.RootSSHAuthorizedKeys.IsNull() {
		args = append(args, "--root-ssh-authorized-keys", data.RootSSHAuthorizedKeys.ValueString())
	}

	if !data.TargetImgref.IsNull() {
		args = append(args, "--target-imgref", data.TargetImgref.ValueString())
	}

	if data.DisableSELinux.ValueBool() {
		args = append(args, "--disable-selinux")
	}

	if !data.Bootloader.IsNull() {
		args = append(args, "--bootloader", data.Bootloader.ValueString())
	}

	args = append(args, rawPath)

	// 3. Run bootc install to-disk --via-loopback
	bootcErr := BootcRun(args)
	if bootcErr != nil {
		_ = os.Remove(rawPath)

		resp.Diagnostics.AddError("bootc install failed", bootcErr.Error())

		return
	}

	// 4. Convert raw → qcow2

	convertCmd := exec.CommandContext(ctx, "qemu-img", "convert",
		"-f", "raw", "-O", "qcow2", rawPath, qcow2Path)

	convertOut, convertErr := convertCmd.CombinedOutput()
	if convertErr != nil {
		resp.Diagnostics.AddError("qemu-img convert failed",
			fmt.Sprintf("%v: %s", convertErr, string(convertOut)))

		return
	}

	// 5. Clean up raw file
	_ = os.Remove(rawPath)

	data.ImagePath = types.StringValue(qcow2Path)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (*ImageResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
}

func (*ImageResource) Update(
	_ context.Context,
	_ resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	resp.Diagnostics.AddError("Update not supported",
		"bootc_image is immutable. Changes require replacement.")
}

func (*ImageResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data ImageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !data.ImagePath.IsNull() {
		_ = os.Remove(data.ImagePath.ValueString())
	}
}
