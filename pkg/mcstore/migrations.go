// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package mcstore

import "embed"

//go:embed migrations/*.sql
var MigrationFS embed.FS
