/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergePodTemplateLabels(t *testing.T) {
	dest := map[string]string{
		"dest-label-1": "dest-label-1-value",
		"dest-label-2": "dest-label-2-value",
	}

	src := map[string]string{
		"dest-label-2": "overrides-dest-label-2-value",
		"src-label-1":  "src-label-1-value",
		"src-label-2":  "src-label-2-value",
	}

	merged := mergePodTemplateLabels(dest, src)

	assert.Len(t, merged, 4)
	assert.Equal(t, merged["dest-label-1"], "dest-label-1-value")
	assert.Equal(t, merged["dest-label-2"], "dest-label-2-value")
	assert.Equal(t, merged["src-label-1"], "src-label-1-value")
	assert.Equal(t, merged["src-label-2"], "src-label-2-value")
}
