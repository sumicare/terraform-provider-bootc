// Copyright 2025 Sumicare
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//! Thin C ABI bridge to bootc-lib for Go/CGo consumption.
//!
//! Exposes a single function `bootc_run` that delegates to
//! `bootc_lib::cli::run_from_iter`, allowing the Go side to
//! invoke any bootc subcommand by constructing the appropriate
//! argument vector.

use std::ffi::{CStr, OsString};
use std::os::unix::ffi::OsStringExt;

/// Run bootc with the given argument vector.
///
/// `argc` is the number of arguments, `argv` is a C-style array of
/// null-terminated UTF-8 strings. The first element should be the
/// program name (e.g. "bootc").
///
/// Returns 0 on success, 1 on error. Errors are printed to stderr.
/// Panics abort the process (release profile uses `panic = "abort"`).
///
/// # Safety
///
/// `argv` must point to `argc` valid, non-null, null-terminated C strings.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn bootc_run(argc: i32, argv: *const *const libc::c_char) -> i32 {
    if argc < 1 || argv.is_null() {
        eprintln!("bootc-bridge: invalid arguments (argc={argc})");
        return 1;
    }

    // SAFETY: caller guarantees argv has argc valid pointers
    let args: Vec<OsString> = unsafe {
        (0..argc as usize)
            .map(|i| {
                let ptr = *argv.add(i);
                let cstr = CStr::from_ptr(ptr);
                OsString::from_vec(cstr.to_bytes().to_vec())
            })
            .collect()
    };

    match run_inner(args) {
        Ok(()) => 0,
        Err(e) => {
            eprintln!("bootc-bridge: {e:#}");
            1
        }
    }
}

#[allow(dead_code)]
fn run_inner(args: Vec<OsString>) -> anyhow::Result<()> {
    bootc_lib::cli::global_init()?;

    let runtime = tokio::runtime::Builder::new_current_thread()
        .enable_all()
        .build()?;

    runtime.block_on(bootc_lib::cli::run_from_iter(args))
}
