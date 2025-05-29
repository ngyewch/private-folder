package main

import (
	"embed"
	"errors"
	"fmt"
	"github.com/adrg/xdg"
	"github.com/go-git/go-git/v5"
	"github.com/pelletier/go-toml/v2"
	"github.com/sethvargo/go-diceware/diceware"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	//go:embed resources
	resourceFS embed.FS
)

type PrivateFolderHelper struct {
	repo *git.Repository
	root string
}

type PrivateFolderConfig struct {
	FolderName string `toml:"folderName"`
}

func NewPrivateFolderHelper() (*PrivateFolderHelper, error) {
	repo, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, err
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	return &PrivateFolderHelper{
		repo: repo,
		root: worktree.Filesystem.Root(),
	}, nil
}

func (helper *PrivateFolderHelper) Init() error {
	baseDir := filepath.Join(helper.root, ".private")
	err := createDir(baseDir, 0755)
	if err != nil {
		return err
	}

	gitignorePath := filepath.Join(baseDir, ".gitignore")
	err = createFile(gitignorePath, func(path string) error {
		return copyResourceFile("resources/gitignore", gitignorePath)
	})
	if err != nil {
		return err
	}

	var config PrivateFolderConfig
	configPath := filepath.Join(baseDir, "config.toml")
	err = createFile(configPath, func(path string) error {
		for {
			folderName, err := generateFolderName()
			if err != nil {
				return err
			}
			targetPrivateFolderPath := filepath.Join(xdg.ConfigHome, "private-folder", folderName)
			_, err = os.Stat(targetPrivateFolderPath)
			if err != nil {
				if os.IsNotExist(err) {
					err = os.MkdirAll(targetPrivateFolderPath, os.ModePerm)
					if err != nil {
						return err
					}
					config.FolderName = folderName
					break
				} else {
					return err
				}
			}
		}

		err = createFile(configPath, func(path string) error {
			return writeConfigFile(config, path)
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = func() error {
		r, err := os.Open(configPath)
		if err != nil {
			return err
		}
		defer func(r *os.File) {
			_ = r.Close()
		}(r)

		tomlDecoder := toml.NewDecoder(r)
		err = tomlDecoder.Decode(&config)
		if err != nil {
			return err
		}

		return nil
	}()
	if err != nil {
		return err
	}

	localPrivateFolderPath := filepath.Join(baseDir, "files")
	err = createDirSymlink(localPrivateFolderPath, func(path string) error {
		targetPrivateFolderPath := filepath.Join(xdg.ConfigHome, "private-folder", config.FolderName)
		err = os.Symlink(targetPrivateFolderPath, path)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func copyResourceFile(src string, dst string) error {
	r, err := resourceFS.Open(src)
	if err != nil {
		return err
	}
	defer func(r fs.File) {
		_ = r.Close()
	}(r)

	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func(w *os.File) {
		_ = w.Close()
	}(w)

	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}

	return nil
}

func writeConfigFile(config PrivateFolderConfig, configPath string) error {
	w, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer func(w *os.File) {
		_ = w.Close()
	}(w)

	tomlEncoder := toml.NewEncoder(w)
	err = tomlEncoder.Encode(config)
	if err != nil {
		return err
	}

	return nil
}

func createDir(dir string, perm os.FileMode) error {
	stat, err := os.Lstat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = os.MkdirAll(dir, perm)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if !stat.IsDir() {
			return fmt.Errorf("%s exists but is not a directory", dir)
		}
	}
	return nil
}

func createFile(path string, create func(path string) error) error {
	stat, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = create(path)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if stat.IsDir() {
			return fmt.Errorf("%s exists but is not a file", path)
		}
	}
	return nil
}

func createDirSymlink(dir string, create func(path string) error) error {
	stat, err := os.Lstat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = create(dir)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if (stat.Mode() & os.ModeSymlink) == 0 {
			return fmt.Errorf("%s exists but is not a symlink", dir)
		}
		stat, err = os.Stat(dir)
		if err != nil {
			return err
		}
		if !stat.IsDir() {
			return fmt.Errorf("%s exists but is not a directory", dir)
		}
	}
	return nil
}

func generateFolderName() (string, error) {
	list, err := diceware.Generate(4)
	if err != nil {
		return "", err
	}
	return strings.Join(list, "-"), nil
}
