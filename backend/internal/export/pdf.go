package export

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"jobcard/internal/domain"
)

var (
	capsOnce sync.Once
	caps     domain.Capabilities
)

// DetectCapabilities reports what this machine can do. PDF needs Microsoft Word (used headless
// to convert the generated .docx); reveal needs Windows Explorer.
func DetectCapabilities() domain.Capabilities {
	capsOnce.Do(func() {
		caps.Docx = true
		if runtime.GOOS == "windows" {
			caps.RevealInFolder = true
			for _, root := range []string{"HKLM", "HKCU"} {
				key := root + `\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\winword.exe`
				if exec.Command("reg", "query", key, "/ve").Run() == nil {
					caps.PDF = true
					break
				}
			}
		}
	})
	return caps
}

const wordScript = `param([string]$in,[string]$out)
$ErrorActionPreference='Stop'
$w = New-Object -ComObject Word.Application
$w.Visible = $false
$w.DisplayAlerts = 0
try {
  $d = $w.Documents.Open($in, $false, $true)
  $d.SaveAs2([string]$out, 17)
  $d.Close($false)
} finally { $w.Quit() }
`

// DocxToPDF converts docx bytes to PDF bytes with headless Word.
func DocxToPDF(docx []byte) ([]byte, error) {
	if !DetectCapabilities().PDF {
		return nil, fmt.Errorf("pdf conversion needs Microsoft Word")
	}
	dir, err := os.MkdirTemp("", "jobcard-pdf-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	in, out, script := filepath.Join(dir, "in.docx"), filepath.Join(dir, "out.pdf"), filepath.Join(dir, "convert.ps1")
	if err := os.WriteFile(in, docx, 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(script, []byte(wordScript), 0o600); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", script, "-in", in, "-out", out)
	if b, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("word conversion failed: %v: %s", err, string(b))
	}
	return os.ReadFile(out)
}

// Reveal opens a folder in the system file manager (Windows Explorer).
func Reveal(path string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("reveal is only supported on Windows")
	}
	// explorer.exe returns a non-zero exit code even on success, so only Start errors matter.
	return exec.Command("explorer.exe", path).Start()
}
