package serviceops

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/podman"
)

// PrepareServiceData validates existing data or seeds an empty directory from
// the image. Nonempty incomplete directories require an explicit recovery.
func PrepareServiceData(name string) error {
	var svc *config.CustomService
	var err error
	if config.IsDefaultPreset(name) {
		p, loadErr := config.LoadPreset(name)
		if loadErr != nil {
			return loadErr
		}
		version := ""
		if cfg, loadErr := config.LoadGlobal(); loadErr == nil {
			version = cfg.Services[name].CanonicalVersion
		}
		if version != "" {
			svc, err = p.ResolvePinned(version)
		} else {
			svc, err = p.Resolve("")
		}
	} else {
		svc, err = config.LoadCustomService(name)
	}
	if err != nil {
		return err
	}
	if len(svc.DataSeed) == 0 {
		return nil
	}
	if image := podman.ServiceImage("lerd-" + name); image != "" {
		svc.Image = image
	}
	return prepareServiceData(svc, podman.CopyImageData)
}

func prepareServiceData(svc *config.CustomService, copyData func(*config.CustomService, string) error) error {
	if len(svc.DataSeed) == 0 {
		return nil
	}
	if err := config.ValidateCustomService(svc); err != nil {
		return err
	}
	defer lockService(svc.Name)()
	dir := config.DataSubDir(svc.Name)
	if st, err := os.Lstat(dir); err == nil && (!st.IsDir() || st.Mode()&os.ModeSymlink != 0) {
		return fmt.Errorf("data directory for %s must be a real directory: %s", svc.Name, dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(entries) > 0 {
		if err := validateDataSeed(dir, svc.DataSeed); err != nil {
			return fmt.Errorf("%s data is incomplete at %s: %w; existing files were preserved; restore a complete backup or explicitly reinstall with --reset-data", svc.Name, dir, err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Dir(dir), "."+svc.Name+"-seed-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err := copyData(svc, stage); err != nil {
		return fmt.Errorf("seeding %s data from %s: %w", svc.Name, svc.Image, err)
	}
	if err := validateDataSeed(stage, svc.DataSeed); err != nil {
		return fmt.Errorf("image %s contains an incomplete data seed: %w", svc.Image, err)
	}
	// Rename replaces only an empty destination, so a concurrent writer cannot
	// have its database overwritten by a copy that began while it was empty.
	if err := os.Rename(stage, dir); err != nil {
		return fmt.Errorf("publishing data seed for %s: %w", svc.Name, err)
	}
	return nil
}

func validateDataSeed(dir string, required []string) error {
	for _, name := range required {
		p := filepath.Join(dir, name)
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			return fmt.Errorf("missing required data file %s: %w", name, err)
		}
		rel, err := filepath.Rel(dir, resolved)
		if err != nil || !filepath.IsLocal(rel) {
			return fmt.Errorf("data file %s points outside the data directory", name)
		}
		st, err := os.Stat(p)
		if err != nil || !st.Mode().IsRegular() || st.Size() == 0 {
			return fmt.Errorf("required data file %s is missing, empty or not a regular file", name)
		}
	}
	return nil
}
