package patcher

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/lthummus/bdpatch/fsutils"
)

const (
	ExtDataOffset       = 0x0C
	ThreeDByteOffset    = 0x2C
	ParentDirectoryName = "AVCHD"
)

type patchFile struct {
	main   string
	backup string
}

var potentialMovieObjects = []string{
	filepath.Join("BDMV", "MovieObject.bdmv"),
	filepath.Join("BDMV", "MOVIEOBJECT.BDM"),
}

var filesToPatch = []patchFile{
	{
		main:   filepath.Join("BDMV", "index.bdmv"),
		backup: filepath.Join("BDMV", "BACKUP", "index.bdmv"),
	},
	{
		main:   filepath.Join("BDMV", "INDEX.BDM"),
		backup: filepath.Join("BDMV", "BACKUP", "INDEX.BDM"),
	},
}

// extData is a 24-byte dummy block of ExtensionData that we will append to the end of the index file
//
// this extension data is well structured, but is otherwise empty
var extData = [24]byte{
	0x00, 0x00, 0x00, 0x18,
	0x00, 0x00, 0x00, 0x18,
	0x00, 0x00, 0x00, 0x01,
	0x10, 0x00, 0x01, 0x00,
	0x00, 0x00, 0x00, 0x18,
	0x00, 0x00, 0x00, 0x00,
}

func validateStructure(path string) error {
	// check to make sure we have a BDMV directory to look at
	// and a BACKUP directory inside

	bdmvDir := filepath.Join(path, "BDMV")
	if !fsutils.Exists(bdmvDir) {
		return fmt.Errorf("BDMV directory does not exist. is this a blu-ray rip?")
	}

	return nil
}

func findMovieObject(path string) (string, error) {
	for _, curr := range potentialMovieObjects {
		p := filepath.Join(path, curr)
		if fsutils.Exists(p) {
			return p, nil
		}
	}

	return "", fmt.Errorf("could not find MovieObject")
}

func findFileToPatch(path string) (string, error) {
	for _, curr := range filesToPatch {
		potentialToPatch := filepath.Join(path, curr.main)
		if !fsutils.Exists(potentialToPatch) {
			continue
		}

		// we're gonna use this file, but first make the corresponding backup if necessary
		backupPath := filepath.Join(path, curr.backup)
		if !fsutils.Exists(backupPath) {
			log.Printf("found %s. making a backup first", potentialToPatch)
			backupDir := filepath.Dir(backupPath)
			if err := os.MkdirAll(backupDir, 0755); err != nil {
				return "", fmt.Errorf("could not create backup directory: %w", err)
			}
			if err := fsutils.CopyFile(potentialToPatch, backupPath); err != nil {
				return "", fmt.Errorf("could not make backup: %w", err)
			}
			log.Printf("backup made")
		}

		return potentialToPatch, nil
	}

	return "", fmt.Errorf("no candidate files found in BDMV")
}

func patchIndex(indexFile string, clear3DByte bool) error {
	f, err := os.OpenFile(indexFile, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("could not open file for patching: %w", err)
	}
	defer f.Close()

	// read extension data pointer (0x0C)
	var extDataAddr [4]byte
	if _, err = f.ReadAt(extDataAddr[:], ExtDataOffset); err != nil {
		return fmt.Errorf("could not read ext_data_start_addr: %w", err)
	}

	if extDataAddr != [4]byte{} {
		// file is already patched, but still honor --force-2d if requested
		log.Printf("looks like we've been here already")
		if clear3DByte {
			log.Printf("clearing 3D byte")
			if _, err = f.WriteAt([]byte{0x00}, ThreeDByteOffset); err != nil {
				return fmt.Errorf("could not clear 3D flag: %w", err)
			}
			if err = f.Sync(); err != nil {
				return err
			}
		}
		return nil
	}

	fileStats, err := f.Stat()
	if err != nil {
		return fmt.Errorf("could not get file length: %w", err)
	}
	fileSize := fileStats.Size()

	// and now write our data at the end
	if _, err = f.WriteAt(extData[:], int64(fileSize)); err != nil {
		return fmt.Errorf("could not append extension data to disc: %w", err)
	}

	// set the pointer inside the index file to point at the end of the file where we'll be writing our data
	var extAddrBE [4]byte
	binary.BigEndian.PutUint32(extAddrBE[:], uint32(fileSize))
	if _, err = f.WriteAt(extAddrBE[:], ExtDataOffset); err != nil {
		return fmt.Errorf("could not write ext_data_start_addr: %w", err)
	}

	if clear3DByte {
		log.Printf("clearing 3D byte")
		if _, err = f.WriteAt([]byte{0x00}, ThreeDByteOffset); err != nil {
			return fmt.Errorf("could not clear 3D flag: %w", err)
		}
	}

	err = f.Sync()
	if err != nil {
		return err
	}

	return nil
}

func fixMovieObjectModificationTime(indexFilePath string, movieObjectPath string) error {
	indexFileStats, err := os.Stat(indexFilePath)
	if err != nil {
		return err
	}

	movieObjectStats, err := os.Stat(movieObjectPath)
	if err != nil {
		return err
	}

	currentMode := movieObjectStats.Mode().Perm()
	readOnly := currentMode&0222 == 0

	if readOnly {
		// if we're read only, we need to fix that to fix the file
		if err = os.Chmod(movieObjectPath, currentMode|0222); err != nil {
			return fmt.Errorf("could not clear readonly flag: %w", err)
		}
	}

	if err = os.Chtimes(movieObjectPath, time.Now(), indexFileStats.ModTime()); err != nil {
		return fmt.Errorf("could not re-set modified time on movie object: %w", err)
	}

	if readOnly {
		// now restore read only state if needed
		if err = os.Chmod(movieObjectPath, currentMode); err != nil {
			return fmt.Errorf("could not re-enable read only: %w", err)
		}

	}

	return nil
}

func restructureDisc(path string) error {
	directories, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("could not read directories that exist: %w", err)
	}

	destDir := filepath.Join(path, ParentDirectoryName)
	if fsutils.Exists(destDir) {
		log.Printf("destination directory %s already exists...doing nothing", destDir)
		return nil
	}

	if err = os.Mkdir(destDir, 0755); err != nil {
		return fmt.Errorf("could not create %s: %w", destDir, err)
	}

	for _, curr := range directories {
		srcPath := filepath.Join(path, curr.Name())
		destPath := filepath.Join(path, ParentDirectoryName, curr.Name())
		err = os.Rename(srcPath, destPath)
		if err != nil {
			return fmt.Errorf("could not rename %s: %w", curr.Name(), err)
		}
	}

	return nil
}

func PatchDisc(ctx context.Context, path string, clear3DFlag bool, noRestructure bool) error {
	err := validateStructure(path)
	if err != nil {
		return err
	}

	fileToPatch, err := findFileToPatch(path)
	if err != nil {
		return err
	}
	log.Printf("found file to patch: %s", fileToPatch)

	log.Printf("finding movie object")
	movieObject, err := findMovieObject(path)
	if err != nil {
		return err
	}
	log.Printf("found movie object %s", movieObject)

	log.Printf("checking to see if we own the movie object (or are root)")
	if !fsutils.CanModifyTimestamp(movieObject) {
		return fmt.Errorf("we do not own the movie object. maybe try running as root?")
	}
	log.Printf("looks good!")

	err = patchIndex(fileToPatch, clear3DFlag)
	if err != nil {
		return err
	}
	log.Printf("patched index file")

	err = fixMovieObjectModificationTime(fileToPatch, movieObject)
	if err != nil {
		return err
	}

	log.Printf("fixed movie object modification time")

	runningInCorrectDir := filepath.Base(path) == ParentDirectoryName

	if !runningInCorrectDir && !noRestructure {
		log.Printf("restructuring for oppo")
		err = restructureDisc(path)
		if err != nil {
			return err
		}
	}

	return nil
}
