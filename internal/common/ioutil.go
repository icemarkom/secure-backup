// Copyright 2026 Marko Milivojevic
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
//
// SPDX-License-Identifier: Apache-2.0

package common

// IOBufferSize is the standard buffer size for all pipeline IO operations.
// Benchmarked across 32KB–4MB; 1MiB chosen for reduced syscall overhead
// on real disk IO while keeping memory usage trivial.
const IOBufferSize = 1024 * 1024 // 1 MiB

// NewBuffer returns a new byte slice of IOBufferSize for use with io.CopyBuffer.
func NewBuffer() []byte {
	return make([]byte, IOBufferSize)
}
