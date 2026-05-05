package downloader

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// pythonDownload uses the Python cloudscraper to download a demo file.
// cloudscraper solves Cloudflare challenges natively.
// HLTV demos come as .rar archives containing the .dem file — we extract the .dem.
func pythonDownload(ctx context.Context, downloadURL, destPath string) error {
	tmpDir, err := os.MkdirTemp("", "hltv-download-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	rarPath := filepath.Join(tmpDir, "demo.rar")

	script := fmt.Sprintf(`
import sys, os
try:
    import cloudscraper
except ImportError:
    print("FATAL: cloudscraper not installed. Run: pip install cloudscraper", file=sys.stderr)
    sys.exit(1)

scraper = cloudscraper.create_scraper(
    browser={"browser": "firefox", "platform": "windows", "mobile": False}
)
url = %q
dest = %q
os.makedirs(os.path.dirname(dest) or ".", exist_ok=True)

response = scraper.get(url, stream=True, allow_redirects=True, timeout=120)
response.raise_for_status()

with open(dest, "wb") as f:
    for chunk in response.iter_content(chunk_size=128*1024):
        if chunk:
            f.write(chunk)
print("OK", flush=True)
`, downloadURL, rarPath)

	scriptPath := filepath.Join(tmpDir, "download.py")
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		return fmt.Errorf("write script: %w", err)
	}

	cmd := exec.CommandContext(ctx, "python3", scriptPath)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("python download: %w", err)
	}
	if string(output) != "OK\n" {
		return fmt.Errorf("python download: unexpected output: %s", string(output))
	}

	info, err := os.Stat(rarPath)
	if err != nil {
		return fmt.Errorf("downloaded file check: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("downloaded file is empty")
	}

	// HLTV demos come as .rar archives. Extract the .dem file inside.
	demPath := filepath.Join(tmpDir, "extracted.dem")
	if err := extractRAR(ctx, rarPath, demPath); err != nil {
		return fmt.Errorf("extract rar: %w", err)
	}

	// Copy the extracted .dem to the final destination.
	src, err := os.Open(demPath)
	if err != nil {
		return fmt.Errorf("open extracted dem: %w", err)
	}
	defer src.Close()

	os.MkdirAll(filepath.Dir(destPath), 0755)
	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer dst.Close()

	if _, err := dst.ReadFrom(src); err != nil {
		return fmt.Errorf("copy dem: %w", err)
	}

	return nil
}

// extractRAR extracts the first .dem or .dem.gz file from a RAR archive.
// Falls back to unrar CLI if available, otherwise returns the raw rar.
func extractRAR(ctx context.Context, rarPath, destPath string) error {
	tmpDir := filepath.Dir(destPath)

	// Try unrar first.
	if unrar, err := exec.LookPath("unrar"); err == nil {
		cmd := exec.CommandContext(ctx, unrar, "e", "-y", "-o+", rarPath, tmpDir+"/")
		cmd.Stderr = os.Stderr
		if extracted := findExtracted(tmpDir, rarPath); extracted != "" {
			return os.Rename(extracted, destPath)
		}
	}

	// Try unar (macOS — already installed via brew).
	if unar, err := exec.LookPath("unar"); err == nil {
		cmd := exec.CommandContext(ctx, unar, "-force-overwrite", "-output-directory", tmpDir, rarPath)
		cmd.Stderr = os.Stderr
		cmd.Output() // unar always exits 0 even on errors; check filesystem
		if extracted := findExtracted(tmpDir, rarPath); extracted != "" {
			return os.Rename(extracted, destPath)
		}
	}

	// Try 7z as fallback.
	if sevenZip, err := exec.LookPath("7z"); err == nil {
		cmd := exec.CommandContext(ctx, sevenZip, "e", "-y", "-o"+tmpDir, rarPath)
		cmd.Stderr = os.Stderr
		cmd.Output()
		if extracted := findExtracted(tmpDir, rarPath); extracted != "" {
			return os.Rename(extracted, destPath)
		}
	}

	return fmt.Errorf("no RAR extractor found (install 'unar' via brew); rar at %s", rarPath)
}

// findExtracted finds the first extractable file (or .dem inside subdir).
func findExtracted(dir, rarPath string) string {
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		name := e.Name()
		if name == filepath.Base(rarPath) || name == "download.py" {
			continue
		}
		if name == "__MACOSX" || filepath.Ext(name) == ".crdownload" {
			continue
		}
		full := filepath.Join(dir, name)
		if e.IsDir() {
			if found := findDemInDir(full); found != "" {
				return found
			}
			continue
		}
		return full
	}
	return ""
}

func findDemInDir(dir string) string {
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() {
			if found := findDemInDir(filepath.Join(dir, e.Name())); found != "" {
				return found
			}
			continue
		}
		if filepath.Ext(e.Name()) == ".dem" {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}
