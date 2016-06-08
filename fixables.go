package ubuntu_sdk_tools

import (
	"github.com/lxc/lxd"
	"github.com/lxc/lxd/shared"
	"os"
	"fmt"
	"path/filepath"
)

type Fixable interface {
	Check(client *lxd.Client) error
	Fix(client *lxd.Client) error
}

type containerAccess struct { }
func (*containerAccess) Check(client *lxd.Client) error {
	targets, err := FindClickTargets(client)
	if err != nil {
		return err
	}

	for _, target := range targets {
		targetPath := shared.VarPath("containers", target.Name)
		fi, err := os.Lstat(targetPath)
		if err != nil {
			return fmt.Errorf("Failed to query container access permissions\n",err)
		}

		if fi.Mode() & os.ModeSymlink == os.ModeSymlink {
			targetPath, err = filepath.EvalSymlinks(targetPath)
			if err != nil {
				return fmt.Errorf("Failed to make rootfs readable. error: %v.\n",err)
			}

			fi, err = os.Lstat(targetPath)
			if err != nil {
				return fmt.Errorf("Failed to query container access permissions\n",err)
			}
		}

		if fi.Mode() != os.ModeDir | LxdContainerPerm {
			return fmt.Errorf("Wrong directory permissions. Container rootfs of %s is not accessible.", target.Name)
		}
	}
	fmt.Println("All containers are accessible.")
	return nil
}
func (*containerAccess) Fix(client *lxd.Client) error {
	fmt.Println("Fixing possible container permission problems....")
	targets, err := FindClickTargets(client)
	if err != nil {
		return err
	}

	for _, target := range targets {
		targetPath := shared.VarPath("containers", target.Name)
		err = os.Chmod(targetPath, LxdContainerPerm)
		if err != nil {
			return fmt.Errorf("Failed to make container readable. error: %v.\n",err)
		}
	}

	fmt.Println("Success.\n")
	return nil
}

var Fixables = []Fixable{
	&containerAccess{},
}
