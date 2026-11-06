package mount

import (
	"golang.org/x/sys/unix"
)

const (
	// ptypes is the set propagation types.
	ptypes = unix.MS_SHARED | unix.MS_PRIVATE | unix.MS_SLAVE | unix.MS_UNBINDABLE

	// pflags is the full set valid flags for a change propagation call.
	pflags = ptypes | unix.MS_REC | unix.MS_SILENT

	// broflags is the combination of bind and read only
	broflags = unix.MS_BIND | unix.MS_RDONLY
)

// isremount returns true if either device name or flags identify a remount request, false otherwise.
func isremount(device string, flags uintptr) bool {
	switch {
	// We treat device "" and "none" as a remount request to provide compatibility with
	// requests that don't explicitly set MS_REMOUNT such as those manipulating bind mounts.
	case flags&unix.MS_REMOUNT != 0, device == "", device == "none":
		return true
	default:
		return false
	}
}

// mount mounts device on target. flags must not contain propagation types;
// those are passed in propFlags and applied in a separate call, as the kernel
// does not allow combining them with other flags. MS_REC in flags means a
// recursive bind (rbind); MS_REC in propFlags means a recursive propagation
// change (e.g. rprivate).
func mount(device, target, mType string, flags, propFlags uintptr, data string) error {
	oflags := flags &^ ptypes
	if !isremount(device, flags) || data != "" {
		// Initial call applying all non-propagation flags for mount
		// or remount with changed data
		if err := unix.Mount(device, target, mType, oflags, data); err != nil {
			return &mountError{
				op:     "mount",
				source: device,
				target: target,
				flags:  oflags,
				data:   data,
				err:    err,
			}
		}
	}

	if propFlags&ptypes != 0 {
		// Change the propagation type.
		if err := unix.Mount("", target, "", propFlags&pflags, ""); err != nil {
			return &mountError{
				op:     "remount",
				target: target,
				flags:  propFlags & pflags,
				err:    err,
			}
		}
	}

	if oflags&broflags == broflags {
		// Remount the bind to apply read only.
		if err := unix.Mount("", target, "", oflags|unix.MS_REMOUNT, ""); err != nil {
			return &mountError{
				op:     "remount-ro",
				target: target,
				flags:  oflags | unix.MS_REMOUNT,
				err:    err,
			}
		}
	}

	return nil
}
