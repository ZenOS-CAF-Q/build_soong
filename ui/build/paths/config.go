// Copyright 2018 Google Inc. All rights reserved.
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

package paths

import "runtime"

type PathConfig struct {
	// Whether to create the symlink in the new PATH for this tool.
	Symlink bool

	// Whether to log about usages of this tool to the soong.log
	Log bool

	// Whether to exit with an error instead of invoking the underlying tool.
	Error bool

	// Whether we use a toybox prebuilt for this tool. Since we don't have
	// toybox for Darwin, we'll use the host version instead.
	Toybox bool
}

var Allowed = PathConfig{
	Symlink: true,
	Log:     false,
	Error:   false,
}

var Forbidden = PathConfig{
	Symlink: false,
	Log:     true,
	Error:   true,
}

var Log = PathConfig{
	Symlink: true,
	Log: true,
	Error: false,
}

// The configuration used if the tool is not listed in the config below.
// Currently this will create the symlink, but log and error when it's used. In
// the future, I expect the symlink to be removed, and this will be equivalent
// to Forbidden.
var Missing = PathConfig{
	Symlink: true,
	Log:     true,
	Error:   true,
}

var Toybox = PathConfig{
	Symlink: false,
	Log:     true,
	Error:   true,
	Toybox:  true,
}

func GetConfig(name string) PathConfig {
	if config, ok := Configuration[name]; ok {
		return config
	}
	return Missing
}

var Configuration = map[string]PathConfig{
	"awk":       Allowed,
	"bash":      Allowed,
	"bc":        Allowed,
	"bzip2":     Allowed,
	"chmod":     Allowed,
	"cp":        Allowed,
	"date":      Allowed,
	"dd":        Allowed,
	"diff":      Allowed,
	"du":        Allowed,
	"echo":      Allowed,
	"egrep":     Allowed,
	"expr":      Allowed,
	"find":      Allowed,
	"fuser":     Allowed,
	"getconf":   Allowed,
	"getopt":    Allowed,
	"git":       Allowed,
	"grep":      Allowed,
	"gzip":      Allowed,
	"hexdump":   Allowed,
	"hostname":  Allowed,
	"jar":       Allowed,
	"java":      Allowed,
	"javap":     Allowed,
	"ln":        Allowed,
	"ls":        Allowed,
	"lsof":      Allowed,
	"m4":        Allowed,
	"md5sum":    Allowed,
	"mktemp":    Allowed,
	"mv":        Allowed,
	"openssl":   Allowed,
	"patch":     Allowed,
	"perl":      Log,
	"pgrep":     Allowed,
	"pkill":     Allowed,
	"ps":        Allowed,
	"pstree":    Allowed,
	"python":    Allowed,
	"python2.7": Allowed,
	"python3":   Allowed,
	"readlink":  Allowed,
	"realpath":  Allowed,
	"rm":        Allowed,
	"rsync":     Allowed,
	"sed":       Allowed,
	"sh":        Allowed,
	"sha1sum":   Allowed,
	"sha256sum": Allowed,
	"sha512sum": Allowed,
	"sort":      Allowed,
	"stat":      Allowed,
	"tar":       Allowed,
	"timeout":   Allowed,
	"tr":        Allowed,
	"unzip":     Allowed,
	"wc":        Allowed,
	"which":     Allowed,
	"xargs":     Allowed,
	"xz":        Allowed,
	"zip":       Allowed,
	"zipinfo":   Allowed,

	// Host toolchain is removed. In-tree toolchain should be used instead.
	// GCC also can't find cc1 with this implementation.
	"ar":         Forbidden,
	"as":         Forbidden,
	"cc":         Forbidden,
	"clang":      Forbidden,
	"clang++":    Forbidden,
	"gcc":        Forbidden,
	"g++":        Forbidden,
	"ld":         Forbidden,
	"ld.bfd":     Forbidden,
	"ld.gold":    Forbidden,
	"pkg-config": Forbidden,

	// On linux we'll use the toybox version of these instead
	"basename": Toybox,
	"cat":      Toybox,
	// TODO (b/121282416): switch back to Toybox when build is hermetic
	"cmp":      Log,
	"comm":     Toybox,
	"cut":      Toybox,
	// TODO (b/121282416): switch back to Toybox when build is hermetic
	"dirname":  Log,
	"env":      Toybox,
	"head":     Toybox,
	"id":       Toybox,
	// TODO (b/121282416): switch back to Toybox when build is hermetic
	"mkdir":    Log,
	"od":       Toybox,
	// TODO (b/121282416): switch back to Toybox when build is hermetic
	"paste":    Log,
	"pwd":      Log,
	"rmdir":    Log,
	"setsid":   Toybox,
	"sleep":    Toybox,
	// TODO (b/121282416): switch back to Toybox when build is hermetic
	"tail":     Log,
	"tee":      Toybox,
	// TODO (b/121282416): switch back to Toybox when build is hermetic
	"touch":    Log,
	"true":     Toybox,
	"uname":    Toybox,
	"uniq":     Toybox,
	"unix2dos": Toybox,
	"whoami":   Toybox,
	// TODO (b/121282416): switch back to Toybox when build is hermetic
	"xxd":      Log,
}

func init() {
	if runtime.GOOS == "darwin" {
		Configuration["md5"] = Allowed
		Configuration["sw_vers"] = Allowed
		Configuration["xcrun"] = Allowed

		// We don't have toybox prebuilts for darwin, so allow the
		// host versions.
		for name, config := range Configuration {
			if config.Toybox {
				Configuration[name] = Allowed
			}
		}
	}
}
