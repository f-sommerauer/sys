//go:build !windows

package mount

import (
	"fmt"
	"strings"
)

var flags = map[string]struct {
	clear bool
	flag  int
}{
	"defaults":      {false, 0},
	"ro":            {false, RDONLY},
	"rw":            {true, RDONLY},
	"suid":          {true, NOSUID},
	"nosuid":        {false, NOSUID},
	"dev":           {true, NODEV},
	"nodev":         {false, NODEV},
	"exec":          {true, NOEXEC},
	"noexec":        {false, NOEXEC},
	"sync":          {false, SYNCHRONOUS},
	"async":         {true, SYNCHRONOUS},
	"dirsync":       {false, DIRSYNC},
	"remount":       {false, REMOUNT},
	"mand":          {false, MANDLOCK},
	"nomand":        {true, MANDLOCK},
	"atime":         {true, NOATIME},
	"noatime":       {false, NOATIME},
	"diratime":      {true, NODIRATIME},
	"nodiratime":    {false, NODIRATIME},
	"bind":          {false, BIND},
	"rbind":         {false, RBIND},
	"unbindable":    {false, UNBINDABLE},
	"runbindable":   {false, RUNBINDABLE},
	"private":       {false, PRIVATE},
	"rprivate":      {false, RPRIVATE},
	"shared":        {false, SHARED},
	"rshared":       {false, RSHARED},
	"slave":         {false, SLAVE},
	"rslave":        {false, RSLAVE},
	"relatime":      {false, RELATIME},
	"norelatime":    {true, RELATIME},
	"strictatime":   {false, STRICTATIME},
	"nostrictatime": {true, STRICTATIME},
}

var validFlags = map[string]bool{
	"":          true,
	"size":      true,
	"mode":      true,
	"uid":       true,
	"gid":       true,
	"nr_inodes": true,
	"nr_blocks": true,
	"mpol":      true,
}

var propagationFlags = map[string]bool{
	"bind":        true,
	"rbind":       true,
	"unbindable":  true,
	"runbindable": true,
	"private":     true,
	"rprivate":    true,
	"shared":      true,
	"rshared":     true,
	"slave":       true,
	"rslave":      true,
}

// MergeTmpfsOptions merge mount options to make sure there is no duplicate.
func MergeTmpfsOptions(options []string) ([]string, error) {
	// We use collisions maps to remove duplicates.
	// For flag, the key is the flag value (the key for propagation flag is -1)
	// For data=value, the key is the data
	flagCollisions := map[int]bool{}
	dataCollisions := map[string]bool{}

	var newOptions []string
	// We process in reverse order
	for i := len(options) - 1; i >= 0; i-- {
		option := options[i]
		if option == "defaults" {
			continue
		}
		if f, ok := flags[option]; ok && f.flag != 0 {
			// There is only one propagation mode
			key := f.flag
			if propagationFlags[option] {
				key = -1
			}
			// Check to see if there is collision for flag
			if !flagCollisions[key] {
				// We prepend the option and add to collision map
				newOptions = append([]string{option}, newOptions...)
				flagCollisions[key] = true
			}
			continue
		}
		opt, _, ok := strings.Cut(option, "=")
		if !ok || !validFlags[opt] {
			return nil, fmt.Errorf("invalid tmpfs option %q", opt)
		}
		if !dataCollisions[opt] {
			// We prepend the option and add to collision map
			newOptions = append([]string{option}, newOptions...)
			dataCollisions[opt] = true
		}
	}

	return newOptions, nil
}

// parseOptions parses fstab type mount options into mount() flags,
// propagation flags, and device specific data.
//
// Propagation options (private, rprivate, shared, rshared, slave, rslave,
// unbindable, runbindable) are returned separately from the other flags,
// because their recursive variants include MS_REC, which has a different
// meaning in the initial mount call (it turns bind into rbind).
func parseOptions(options string) (flag, propFlag int, data string) {
	var dataOpts []string

	for _, o := range strings.Split(options, ",") {
		// If the option does not exist in the flags table or the flag
		// is not supported on the platform,
		// then it is a data value for a specific fs type
		f, exists := flags[o]
		if !exists || f.flag == 0 {
			dataOpts = append(dataOpts, o)
			continue
		}
		target := &flag
		if isPropagationOption(o) {
			target = &propFlag
		}
		if f.clear {
			*target &= ^f.flag
		} else {
			*target |= f.flag
		}
	}
	return flag, propFlag, strings.Join(dataOpts, ",")
}

// isPropagationOption reports whether o changes the propagation type of a
// mount. bind and rbind are not propagation options.
func isPropagationOption(o string) bool {
	return propagationFlags[o] && o != "bind" && o != "rbind"
}
