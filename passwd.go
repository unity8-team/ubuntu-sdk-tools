// +build darwin freebsd linux
// +build cgo
package ubuntu_sdk_tools

/*
#define _GNU_SOURCE
#include <unistd.h>
#include <sys/types.h>
#include <pwd.h>
#include <stdlib.h>
#include <errno.h>
#include <grp.h>
#include <stdio.h>
#include <shadow.h>

size_t groupMemberCount (struct group *grp)
{
    if (grp == NULL) return 0;
    size_t idx = 0;
    while (1) {
    	if(grp->gr_mem[idx] == NULL)
    	    break;
         idx++;
    }
    return idx;
}

*/
import "C"
import (
	"unsafe"
	"fmt"
	"syscall"
)

type Passwd struct {
	Uid uint32
	Gid uint32
	Dir string
	Shell string
	LoginName string
}

func Getpwnam(username string) (*Passwd, error) {
	var pwd C.struct_passwd
	var result *C.struct_passwd

	bufSize := C.sysconf(C._SC_GETPW_R_SIZE_MAX)
	if bufSize <= 0 || bufSize > 1<<20 {
		return nil, fmt.Errorf("unreasonable _SC_GETPW_R_SIZE_MAX of %d", bufSize)
	}

	buf := C.malloc(C.size_t(bufSize))
	defer C.free(buf)
	var rv C.int

	nameC := C.CString(username)
	defer C.free(unsafe.Pointer(nameC))

	rv, err := C.getpwnam_r(nameC,
		&pwd,
		(*C.char)(buf),
		C.size_t(bufSize),
		&result)
	if rv != 0 {
		return nil, fmt.Errorf("error while looking up username %s: %s", username, syscall.Errno(rv))
	}
	if result == nil {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("Unknown error while getting the user entry for %s", username)
	}

	return &Passwd{
		Uid: uint32(pwd.pw_uid),
		Gid: uint32(pwd.pw_uid),
		Dir: C.GoString(pwd.pw_dir),
		Shell: C.GoString(pwd.pw_shell),
		LoginName: C.GoString(pwd.pw_name)}, nil
}

type GroupEntry struct {
	Gid uint32
	Name string
	Members []string
}

func GetGroups() ([]GroupEntry, error){
	var grp C.struct_group
	var result *C.struct_group
	allGroups := make([]GroupEntry, 0)

	//_SC_GETGR_R_SIZE_MAX
	bufSize := C.sysconf(C._SC_GETGR_R_SIZE_MAX)
	if bufSize <= 0 || bufSize > 1<<20 {
		return nil, fmt.Errorf("unreasonable _SC_GETPW_R_SIZE_MAX of %d", bufSize)
	}
	buf := C.malloc(C.size_t(bufSize))
	defer C.free(buf)

	for {
		result = nil
		res, err := C.getgrent_r(&grp, (*C.char)(buf), C.size_t(bufSize), &result)
		if res != 0 {
			if err != nil {
				return nil, err
			}
			break
		}

		currEntry := GroupEntry{}
		currEntry.Gid = uint32(result.gr_gid)
		currEntry.Name = C.GoString(result.gr_name)

		groupMemberCnt := C.groupMemberCount(result)
		memberList := ((*[1 << 30]*C.char)(unsafe.Pointer(result.gr_mem)))[:groupMemberCnt]

		for _, val := range memberList {
			currEntry.Members = append(currEntry.Members, C.GoString(val))
		}
		allGroups = append(allGroups, currEntry)
	}

	return allGroups, nil
}

type SPasswd struct {
	Sp_namp string   /* Login name */
	Sp_pwdp string   /* Encrypted password */
}

func Getspnam (username string) (*SPasswd, error) {
	var spwd C.struct_spwd
	var result *C.struct_spwd

	bufSize := C.sysconf(C._SC_GETPW_R_SIZE_MAX)
	if bufSize <= 0 || bufSize > 1<<20 {
		return nil, fmt.Errorf("unreasonable _SC_GETPW_R_SIZE_MAX of %d", bufSize)
	}

	buf := C.malloc(C.size_t(bufSize))
	defer C.free(buf)
	var rv C.int

	nameC := C.CString(username)
	defer C.free(unsafe.Pointer(nameC))

	rv, err := C.getspnam_r(nameC,
		&spwd,
		(*C.char)(buf),
		C.size_t(bufSize),
		&result)
	if rv != 0 {
		return nil, fmt.Errorf("error while looking up passwd for %s: %s", username, syscall.Errno(rv))
	}
	if result == nil {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("Unknown error while getting the password entry for %s", username)
	}

	return &SPasswd{
		Sp_namp:C.GoString(result.sp_namp),
		Sp_pwdp:C.GoString(result.sp_pwdp),
	}, nil
}
