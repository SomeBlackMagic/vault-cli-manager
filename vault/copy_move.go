package vault

import (
	"fmt"
	"os"
	"strings"

	"github.com/jhunt/go-ansi"
)

type MoveCopyOpts struct {
	SkipIfExists bool
	Quiet        bool
	//Deep copies all versions and overwrites all versions at the target location
	Deep bool
	//DeletedVersions undeletes, reads, and redeletes the deleted keys
	// It also puts in dummy destroyed keys to dest to match destroyed keys from src
	//Makes no sense without Deep
	DeletedVersions bool
}

// Copy copies secrets from one path to another.
// With a secret:key specified: key -> key is good.
// key -> no-key is okay - we assume to keep old key name
// no-key -> key is bad. That makes no sense and the user should feel bad.
// Returns KeyNotFoundError if there is no such specified key in the secret at oldpath
func (v *Vault) Copy(oldpath, newpath string, opts MoveCopyOpts) error {
	oldpath = Canonicalize(oldpath)
	newpath = Canonicalize(newpath)

	if opts.DeletedVersions && !opts.Deep {
		panic("Gave DeletedVersions and not Deep")
	}
	var err error
	reqState := verifyStateAlive
	if opts.DeletedVersions {
		reqState = verifyStateAliveOrDeleted
	}

	err = v.verifySecretState(oldpath, verifyOpts{
		State:      reqState,
		AnyVersion: opts.Deep,
	})
	if err != nil {
		return err
	}

	if opts.SkipIfExists {
		if _, err := v.Read(newpath); err == nil {
			if !opts.Quiet {
				ansi.Fprintf(os.Stderr, "@R{Cowardly refusing to copy/move data into} @C{%s}@R{, as that would clobber existing data}\n", newpath)
			}
			return nil
		} else if !IsNotFound(err) {
			return err
		}
	}

	srcPath, srcKey, srcVersion := ParsePath(oldpath)
	dstPath, dstKey, dstVersion := ParsePath(newpath)

	if dstVersion != 0 {
		return fmt.Errorf("Copying a secret to a specific destination version is not supported")
	}

	if opts.Deep && srcVersion != 0 {
		return fmt.Errorf("Performing a deep copy of a specified version is not supported")
	}

	var toWrite []*Secret
	if srcKey != "" { //Just a single key.
		if opts.Deep {
			return fmt.Errorf("Cannot take deep copy of a specific key")
		}
		srcSecret, err := v.Read(oldpath)
		if err != nil {
			return err
		}

		if !srcSecret.Has(srcKey) {
			return NewKeyNotFoundError(oldpath, srcKey)
		}

		if dstKey == "" {
			dstKey = srcKey
		}

		dstOrig, err := v.Read(dstPath)
		if err != nil && !IsSecretNotFound(err) {
			return err
		}

		if IsSecretNotFound(err) {
			dstOrig = NewSecret()
		}

		toWrite = append(toWrite, dstOrig)
		toWrite[0].Set(dstKey, srcSecret.Get(srcKey), false)
	} else {
		if dstKey != "" {
			return fmt.Errorf("Cannot move full secret `%s` into specific key `%s`", oldpath, newpath)
		}
		t, err := v.ConstructSecrets(srcPath, TreeOpts{
			FetchKeys:           true,
			GetOnly:             true,
			FetchAllVersions:    opts.Deep || srcVersion != 0,
			GetDeletedVersions:  opts.Deep && opts.DeletedVersions,
			AllowDeletedSecrets: opts.Deep || srcVersion != 0,
		})

		if err != nil {
			return err
		}

		if len(t) == 0 {
			// Prevent a panic
			return NewSecretNotFoundError(srcPath)
		}

		if srcVersion != 0 {
			//Filter results to the specific requested secret
			for i := range t[0].Versions {
				if t[0].Versions[i].Number == uint(srcVersion) {
					t[0].Versions = []SecretVersion{t[0].Versions[i]}
					break
				}
			}
		}

		err = t[0].Copy(v, dstPath, TreeCopyOpts{Clear: opts.Deep, Pad: opts.Deep})
		if err != nil {
			return err
		}
	}

	for i := range toWrite {
		err := v.Write(dstPath, toWrite[i])
		if err != nil {
			return err
		}
	}

	return nil
}

// MoveCopyTree will recursively copy all nodes from the root to the new location.
// This function will get confused about 'secret:key' syntax, so don't let those
// get routed here - they don't make sense for a recursion anyway.
func (v *Vault) MoveCopyTree(oldRoot, newRoot string, f func(string, string, MoveCopyOpts) error, opts MoveCopyOpts) error {
	oldRoot = Canonicalize(oldRoot)
	newRoot = Canonicalize(newRoot)

	tree, err := v.ConstructSecrets(oldRoot, TreeOpts{FetchKeys: false, AllowDeletedSecrets: opts.Deep, SkipVersionInfo: true})
	if err != nil {
		return err
	}
	if opts.SkipIfExists {
		//Writing one secret over a deleted secret isn't clobbering. Completely overwriting a set of deleted secrets would be
		newTree, err := v.ConstructSecrets(newRoot, TreeOpts{FetchKeys: false, AllowDeletedSecrets: !opts.Deep, SkipVersionInfo: true})
		if err != nil && !IsNotFound(err) {
			return err
		}
		existing := map[string]bool{}
		for _, path := range newTree.Paths() {
			existing[path] = true
		}
		existingPaths := []string{}
		for _, path := range tree.Paths() {
			newPath := strings.Replace(path, oldRoot, newRoot, 1)
			if existing[newPath] {
				existingPaths = append(existingPaths, newPath)
			}
		}
		if len(existingPaths) > 0 {
			if !opts.Quiet {
				ansi.Fprintf(os.Stderr, "@R{Cowardly refusing to copy/move data into} @C{%s}@R{, as the following paths would be clobbered:}\n", newRoot)
				for _, path := range existingPaths {
					ansi.Fprintf(os.Stderr, "@R{- }@C{%s}\n", path)
				}
			}
			return nil
		}
	}
	for _, path := range tree.Paths() {
		newPath := strings.Replace(path, oldRoot, newRoot, 1)
		err = f(path, newPath, opts)
		if err != nil {
			return err
		}
	}

	if _, err := v.Read(oldRoot); !IsNotFound(err) { // run through a copy unless we successfully got a 404 from this node
		return f(oldRoot, newRoot, opts)
	}
	return nil
}

// Move moves secrets from one path to another.
// A move is semantically a copy and then a deletion of the original item. For
// more information on the behavior of Move pertaining to keys, look at Copy.
func (v *Vault) Move(oldpath, newpath string, opts MoveCopyOpts) error {
	oldpath = Canonicalize(oldpath)
	newpath = Canonicalize(newpath)

	err := v.canSemanticallyDelete(oldpath)
	if err != nil {
		return fmt.Errorf("Can't move `%s': %s. Did you mean cp?", oldpath, err)
	}
	if err != nil {
		return err
	}

	err = v.Copy(oldpath, newpath, opts)
	if err != nil {
		return err
	}

	if opts.Deep && opts.DeletedVersions {
		err = v.client.DestroyAll(oldpath)
	} else {
		err = v.Delete(oldpath, DeleteOpts{})
		if err != nil {
			return err
		}
	}
	return nil
}
