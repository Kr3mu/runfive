package fxserver

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"golang.org/x/net/html"
)

const (
	windowsArtifactsURL = "https://runtime.fivem.net/artifacts/fivem/build_server_windows/master/"
	linuxArtifactsURL   = "https://runtime.fivem.net/artifacts/fivem/build_proot_linux/master/"
)

// Client lists and resolves upstream FiveM server artifacts for a single host OS.
type Client struct {
	hostOS  string
	baseURL string
	fetch   func(context.Context, string) ([]byte, error)
}

// NewClient creates a scraper client for the current host OS.
func NewClient() (*Client, error) {
	return NewClientForOS(runtime.GOOS)
}

// NewClientForOS creates a scraper client for the provided Go host OS.
func NewClientForOS(hostOS string) (*Client, error) {
	baseURL, err := baseURLForOS(hostOS)
	if err != nil {
		return nil, err
	}

	return &Client{
		hostOS:  hostOS,
		baseURL: baseURL,
		fetch:   fetchPage,
	}, nil
}

// HostOS returns the OS tree managed by this client.
func (c *Client) HostOS() string {
	return c.hostOS
}

// ArchiveExtension returns the archive extension used for this OS.
func (c *Client) ArchiveExtension() string {
	if c.hostOS == "windows" {
		return ".zip"
	}
	return ".tar.xz"
}

// ArchiveName returns the archive filename used for this OS.
func (c *Client) ArchiveName() string {
	if c.hostOS == "windows" {
		return "server.zip"
	}
	return "fx.tar.xz"
}

// ListVersions fetches and returns all upstream artifact versions, newest first.
func (c *Client) ListVersions(ctx context.Context) ([]string, error) {
	body, err := c.fetch(ctx, c.baseURL)
	if err != nil {
		return nil, err
	}

	tags := extractArtifactTags(body)
	seen := make(map[int]bool)
	nums := make([]int, 0, len(tags))
	for _, tag := range tags {
		version, _, _ := strings.Cut(tag, "-")
		n, err := strconv.Atoi(version)
		if err == nil && !seen[n] {
			seen[n] = true
			nums = append(nums, n)
		}
	}

	sort.Sort(sort.Reverse(sort.IntSlice(nums)))

	versions := make([]string, 0, len(nums))
	for _, n := range nums {
		versions = append(versions, strconv.Itoa(n))
	}

	if len(versions) == 0 {
		snippet := string(body)
		if len(snippet) > 240 {
			snippet = snippet[:240]
		}
		return nil, fmt.Errorf("fxserver: no versions found in upstream listing (body starts with %q)", snippet)
	}

	return versions, nil
}

// ResolveTag expands a numeric version to the full upstream directory tag.
func (c *Client) ResolveTag(ctx context.Context, version string) (string, error) {
	body, err := c.fetch(ctx, c.baseURL)
	if err != nil {
		return "", err
	}

	for _, tag := range extractArtifactTags(body) {
		if strings.HasPrefix(tag, version+"-") {
			return tag, nil
		}
	}

	return "", fmt.Errorf("fxserver: version %s not found", version)
}

// DownloadURL returns the archive download URL for a fully resolved upstream tag.
func (c *Client) DownloadURL(tag string) string {
	return fmt.Sprintf("%s%s/%s", c.baseURL, tag, c.ArchiveName())
}

func baseURLForOS(hostOS string) (string, error) {
	switch hostOS {
	case "windows":
		return windowsArtifactsURL, nil
	case "linux":
		return linuxArtifactsURL, nil
	default:
		return "", fmt.Errorf("fxserver: unsupported host OS %q", hostOS)
	}
}

func fetchPage(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fxserver: build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; RunFive/1.0; +https://github.com/Kr3mu/runfive)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, br")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("fxserver: fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fxserver: fetch %s: unexpected status %d", url, resp.StatusCode)
	}

	reader, err := decodeBody(resp)
	if err != nil {
		return nil, fmt.Errorf("fxserver: decode %s: %w", url, err)
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("fxserver: read %s: %w", url, err)
	}

	return body, nil
}

func decodeBody(resp *http.Response) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding"))) {
	case "", "identity":
		return resp.Body, nil
	case "gzip":
		return gzip.NewReader(resp.Body)
	case "br":
		return brotli.NewReader(resp.Body), nil
	default:
		return nil, fmt.Errorf("unsupported content encoding %q", resp.Header.Get("Content-Encoding"))
	}
}

var artifactTagPattern = regexp.MustCompile(`\b(\d+-[0-9a-fA-F]{8,})\b`)

func extractArtifactTags(body []byte) []string {
	seen := make(map[string]struct{})
	tags := make([]string, 0)

	addTag := func(value string) {
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `/`)
		if value == "" {
			return
		}

		if parsed, err := url.Parse(value); err == nil {
			value = parsed.Path
		}
		value = strings.Trim(value, `/`)
		if value == "" {
			return
		}

		for _, segment := range strings.Split(value, "/") {
			if matches := artifactTagPattern.FindStringSubmatch(segment); len(matches) == 2 {
				tag := matches[1]
				if _, ok := seen[tag]; ok {
					return
				}
				seen[tag] = struct{}{}
				tags = append(tags, tag)
				return
			}
		}
	}

	tokenizer := html.NewTokenizer(bytes.NewReader(body))
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			sortTags(tags)
			return tags
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			if token.Data != "a" {
				continue
			}
			for _, attr := range token.Attr {
				if attr.Key == "href" {
					addTag(attr.Val)
				}
			}
		case html.TextToken:
			addTag(string(tokenizer.Text()))
		}
	}
}

func sortTags(tags []string) {
	sort.Slice(tags, func(i, j int) bool {
		leftVer, _, _ := strings.Cut(tags[i], "-")
		rightVer, _, _ := strings.Cut(tags[j], "-")

		leftNum, leftErr := strconv.Atoi(leftVer)
		rightNum, rightErr := strconv.Atoi(rightVer)
		if leftErr == nil && rightErr == nil && leftNum != rightNum {
			return leftNum > rightNum
		}
		return tags[i] > tags[j]
	})
}
