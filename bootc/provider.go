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

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &BootcProvider{}

const providerTypeName = "bootc"

// BootcProvider is the top-level Terraform provider.
type BootcProvider struct {
	version string
}

// New returns a provider factory stamped with the given version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &BootcProvider{version: version}
	}
}

func (p *BootcProvider) Metadata(
	ctx context.Context,
	req provider.MetadataRequest,
	resp *provider.MetadataResponse,
) {
	resp.TypeName = "bootc"
	resp.Version = p.version
}

func (*BootcProvider) Schema(
	ctx context.Context,
	req provider.SchemaRequest,
	resp *provider.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Build qcow2 disk images from bootc container images.",
	}
}

func (*BootcProvider) Configure(
	_ context.Context,
	_ provider.ConfigureRequest,
	_ *provider.ConfigureResponse,
) {
}

func (*BootcProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewImageResource,
	}
}

func (*BootcProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
