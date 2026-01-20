// Package pngutil renders HTML to PNG via an external headless browser.
package pngutil

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultWidth  = 1600
	defaultHeight = 1200
)

func parseSize() (int, int) {
	value := os.Getenv("SNAP_PNG_SIZE")
	if value == "" {
		return defaultWidth, defaultHeight
	}
	parts := strings.Split(value, "x")
	if len(parts) != 2 {
		parts = strings.Split(value, "X")
	}
	if len(parts) != 2 {
		return defaultWidth, defaultHeight
	}
	w, err := strconv.Atoi(parts[0])
	if err != nil || w <= 0 {
		return defaultWidth, defaultHeight
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil || h <= 0 {
		return defaultWidth, defaultHeight
	}
	return w, h
}

func findTool() (string, error) {
	candidates := []string{
		"google-chrome",
		"chrome",
		"msedge",
		"chromium",
		"chromium-browser",
		"wkhtmltoimage",
	}
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("png tool not found (chromium/chrome/msedge/wkhtmltoimage)")
}

func writeFile(path, contents string) error {
	return os.WriteFile(path, []byte(contents), 0o644)
}

func fileURL(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	return u.String()
}

func runChromium(cmd string, htmlPath, outPath string, width, height int, userDataDir string) error {
	profileDir := filepath.Join(userDataDir, "profile")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		return err
	}
	args := []string{
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--no-first-run",
		"--disable-extensions",
		"--disable-dev-shm-usage",
		"--disable-crash-reporter",
		"--disable-breakpad",
		"--crash-dumps-dir=" + userDataDir,
		"--disable-features=Translate,Crashpad",
		"--user-data-dir=" + profileDir,
		"--window-size=" + fmt.Sprintf("%d,%d", width, height),
		"--screenshot=" + outPath,
		fileURL(htmlPath),
	}
	command := exec.Command(cmd, args...)
	command.Env = append(os.Environ(),
		"HOME="+userDataDir,
		"XDG_DATA_HOME="+userDataDir,
		"XDG_CONFIG_HOME="+userDataDir,
	)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func runWkhtml(cmd string, htmlPath, outPath string, width int) error {
	args := []string{
		"--width",
		strconv.Itoa(width),
		"--disable-smart-width",
		htmlPath,
		outPath,
	}
	command := exec.Command(cmd, args...)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// WritePNGFromHTML renders HTML to PNG using a headless browser.
func WritePNGFromHTML(html, outPath string) error {
	tool, err := findTool()
	if err != nil {
		return err
	}
	width, height := parseSize()
	tmpFile, err := os.CreateTemp("", "snap-html-*.html")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := writeFile(tmpPath, html); err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	if filepath.Base(tool) == "wkhtmltoimage" {
		return runWkhtml(tool, tmpPath, outPath, width)
	}

	userDir, err := os.MkdirTemp("", "snap-chrome-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(userDir)

	return runChromium(tool, tmpPath, outPath, width, height, userDir)
}
