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
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type stringOneOfValidator struct {
	values []string
}

func (v stringOneOfValidator) Description(_ context.Context) string {
	return "value must be one of: " + strings.Join(v.values, ", ")
}

func (v stringOneOfValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v stringOneOfValidator) ValidateString(
	_ context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	val := req.ConfigValue.ValueString()
	if slices.Contains(v.values, val) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid value",
		fmt.Sprintf("Expected one of: %s, got: %s", strings.Join(v.values, ", "), val),
	)
}

func stringOneOf(values ...string) validator.String {
	return stringOneOfValidator{values: values}
}
