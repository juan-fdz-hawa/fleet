package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func desktop() *cli.Command {
	return &cli.Command{
		Name:        "desktop",
		Aliases:     nil,
		Usage:       "Creates the Fleet Desktop Application",
		Description: "The desktop command allows the creation and packaging of the Fleet Desktop Application from source.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "platform",
				Usage:    "Target platform (currently supports \"macos\" only)",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Log detailed information when building the application",
			},
		},
		Action: func(c *cli.Context) error {
			if c.String("platform") != "macos" {
				return fmt.Errorf("unsupported platform: %s", c.String("platform"))
			}

			if !c.Bool("verbose") {
				zlog.Logger = zerolog.Nop()
			}

			if err := createMacOSApp(); err != nil {
				return err
			}

			return nil
		},
	}
}

const infoPList = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>fleet-desktop</string>
	<key>CFBundleIdentifier</key>
	<string>com.fleetdm.desktop</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>fleet-desktop</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>0.0.1</string>
	<key>CFBundleVersion</key>
	<string>0.0.1</string>
	<!-- avoid having a blurry icon and text -->
	<key>NSHighResolutionCapable</key>
	<string>True</string>
	<!-- avoid showing the app on the Dock -->
	<key>LSUIElement</key>
	<string>1</string>
</dict>
</plist>
`

func createMacOSApp() error {
	if runtime.GOOS != "darwin" {
		return errors.New("The \"Fleet Desktop\" macOS app can only be created from macOS")
	}

	const appDir = "Fleet Desktop.app"
	defer os.RemoveAll(appDir)

	contentsDir := filepath.Join(appDir, "Contents")
	macOSDir := filepath.Join(contentsDir, "MacOS")
	for _, dir := range []string{macOSDir} {
		if err := secure.MkdirAll(dir, constant.DefaultDirMode); err != nil {
			return fmt.Errorf("create directories: %w", err)
		}
	}

	infoFile := filepath.Join(contentsDir, "Info.plist")
	ioutil.WriteFile(infoFile, []byte(infoPList), 0o644)

	cmd := exec.Command("go", "build", "-o", filepath.Join(macOSDir, "fleet-desktop"), "./"+filepath.Join("orbit", "cmd", "desktop"))
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	zlog.Info().Str("command", cmd.String()).Msg("Build fleet-desktop executable")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compile desktop application: %w", err)
	}

	const tarGzName = "desktop.app.tar.gz"
	if err := compressDir(tarGzName, appDir); err != nil {
		return fmt.Errorf("compress app: %w", err)
	}
	fmt.Printf("Generated %s successfully.\n", tarGzName)

	return nil
}

func compressDir(outPath, dirPath string) error {
	out, err := secure.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, defaultFileMode)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	if err := filepath.Walk(dirPath, func(file string, fi os.FileInfo, err error) error {
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		// From https://golang.org/src/archive/tar/common.go?#L626
		//
		//	"Since fs.FileInfo's Name method only returns the base name of
		// 	the file it describes, it may be necessary to modify Header.Name
		// 	to provide the full path name of the file."
		header.Name = filepath.ToSlash(file)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if !fi.IsDir() {
			f, err := os.Open(file)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("close tar: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("close gzip: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	return nil
}
