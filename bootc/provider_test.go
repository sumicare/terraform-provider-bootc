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
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &BootcProvider{}

func TestBootcProvider_New(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{testVersionDev, testVersionDev},
		{"release", "1.0.0"},
		{"empty", ""},
	}

	for idx := range tests {
		testCase := tests[idx]
		t.Run(testCase.name, func(t *testing.T) {
			factory := New(testCase.version)
			if factory == nil {
				t.Fatal("expected non-nil factory")
			}

			prov := factory()
			if prov == nil {
				t.Fatal("expected non-nil provider")
			}

			bp, ok := prov.(*BootcProvider)
			if !ok {
				t.Fatalf("expected *BootcProvider, got %T", prov)
			}

			if bp.version != testCase.version {
				t.Errorf("version = %q, want %q", bp.version, testCase.version)
			}
		})
	}
}

const (
	testVersionDev = "dev"
)

func TestBootcProvider_Metadata(t *testing.T) {
	prov := &BootcProvider{version: "1.2.3"}
	resp := &provider.MetadataResponse{}

	prov.Metadata(t.Context(), provider.MetadataRequest{}, resp)

	if resp.TypeName != providerTypeName {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, providerTypeName)
	}

	if resp.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", resp.Version, "1.2.3")
	}
}

func TestBootcProvider_Schema(t *testing.T) {
	prov := &BootcProvider{}
	resp := &provider.SchemaResponse{}

	prov.Schema(t.Context(), provider.SchemaRequest{}, resp)

	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}
}

func TestBootcProvider_Configure(t *testing.T) {
	prov := &BootcProvider{}
	resp := &provider.ConfigureResponse{}

	prov.Configure(t.Context(), provider.ConfigureRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected errors: %v", resp.Diagnostics.Errors())
	}
}

func TestBootcProvider_Resources(t *testing.T) {
	prov := &BootcProvider{}
	resources := prov.Resources(t.Context())

	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}

	r := resources[0]()
	resp := &resource.MetadataResponse{}
	r.Metadata(t.Context(), resource.MetadataRequest{ProviderTypeName: providerTypeName}, resp)

	if resp.TypeName != "bootc_image" {
		t.Errorf("resource type = %q, want %q", resp.TypeName, "bootc_image")
	}
}

func TestBootcProvider_DataSources(t *testing.T) {
	prov := &BootcProvider{}
	ds := prov.DataSources(t.Context())

	if ds != nil {
		t.Errorf("expected nil data sources, got %d", len(ds))
	}
}

func TestBootcProvider_Implements(t *testing.T) {
	tests := []struct {
		fn   func() provider.Provider
		name string
	}{
		{New(testVersionDev), testVersionDev},
		{New("1.0.0"), "release"},
	}

	for idx := range tests {
		testCase := tests[idx]
		t.Run(testCase.name, func(t *testing.T) {
			prov := testCase.fn()

			metaResp := &provider.MetadataResponse{}
			prov.Metadata(t.Context(), provider.MetadataRequest{}, metaResp)

			schemaResp := &provider.SchemaResponse{}
			prov.Schema(t.Context(), provider.SchemaRequest{}, schemaResp)

			confResp := &provider.ConfigureResponse{}
			prov.Configure(t.Context(), provider.ConfigureRequest{}, confResp)

			_ = prov.Resources(t.Context())

			ds := prov.DataSources(t.Context())
			if ds != nil {
				t.Error("expected nil data sources")
			}
		})
	}
}
