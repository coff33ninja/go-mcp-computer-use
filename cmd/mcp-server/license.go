package main

import (
	_ "embed"
)

// LICENSE.txt and NOTICE.txt are kept in sync with the repo-root LICENSE and
// NOTICE by scripts/build.ps1 (RefeshLegalFiles) before each build, so the
// binary embeds the current license and attribution without duplication.

//go:embed LICENSE.txt
var embeddedLicense string

//go:embed NOTICE.txt
var embeddedNotice string
