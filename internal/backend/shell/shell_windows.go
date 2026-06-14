//go:build windows

package shell

func defaultShellArgs() []string {
	return []string{"powershell", "-Command"}
}
