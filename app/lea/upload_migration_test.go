package lea_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Ensure no Go source still writes user uploads under public/upload.
// Legacy reads/compat checks (HasPrefix public/upload, Static.Serve) are allowed.
func TestNoGoWritesToPublicUpload(t *testing.T) {
	root := findRepoRoot(t)
	writePattern := regexp.MustCompile(`(?:=|\+)\s*"public/upload/`)
	allowedFiles := map[string]bool{
		// ExportTheme still accepts old theme.Path prefixes for compatibility.
		filepath.Join("app", "service", "ThemeService.go"): true,
	}

	err := filepath.Walk(filepath.Join(root, "app"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		if !strings.Contains(content, "public/upload") {
			return nil
		}
		if allowedFiles[rel] {
			// Allowed file may still contain write assignments; check lines.
			for _, line := range strings.Split(content, "\n") {
				trim := strings.TrimSpace(line)
				if strings.HasPrefix(trim, "//") {
					continue
				}
				if writePattern.MatchString(line) && !strings.Contains(line, "HasPrefix") {
					t.Errorf("%s still assigns public/upload path: %s", rel, trim)
				}
			}
			return nil
		}
		for _, line := range strings.Split(content, "\n") {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "//") {
				continue
			}
			if strings.Contains(line, "public/upload") {
				t.Errorf("%s still references public/upload: %s", rel, trim)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	routes := filepath.Join(root, "conf", "routes")
	data, err := ioutil.ReadFile(routes)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `Static.Serve("files/upload")`) {
		t.Fatalf("conf/routes missing Static.Serve for files/upload")
	}
	if !strings.Contains(text, `/files/upload/*filepath`) {
		t.Fatalf("conf/routes missing /files/upload route")
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "conf", "routes")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("cannot find repo root from %s", wd)
	return ""
}
