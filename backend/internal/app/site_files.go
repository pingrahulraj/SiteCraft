package app

import (
	"os"
	"path/filepath"
	"strings"
)

func siteRootDir(root, tenantID, siteID string) string {
	return filepath.Join(root, "sites", tenantID, siteID)
}

func siteWorkspaceDir(root, tenantID, siteID string) string {
	return filepath.Join(siteRootDir(root, tenantID, siteID), "workspace")
}

func sitePublishedDir(root, tenantID, siteID string) string {
	return filepath.Join(siteRootDir(root, tenantID, siteID), "published")
}

func siteReleasesDir(root, tenantID, siteID string) string {
	return filepath.Join(siteRootDir(root, tenantID, siteID), "releases")
}

func siteReleaseDir(root, tenantID, siteID, releaseID string) string {
	return filepath.Join(siteReleasesDir(root, tenantID, siteID), releaseID)
}

func siteWorkspaceFile(root, tenantID, siteID, relPath string) string {
	return filepath.Join(siteWorkspaceDir(root, tenantID, siteID), filepath.FromSlash(strings.TrimPrefix(relPath, "/")))
}

func sitePublishedFile(root, tenantID, siteID, relPath string) string {
	return filepath.Join(sitePublishedDir(root, tenantID, siteID), filepath.FromSlash(strings.TrimPrefix(relPath, "/")))
}

func siteReleaseFile(root, tenantID, siteID, releaseID, relPath string) string {
	return filepath.Join(siteReleaseDir(root, tenantID, siteID, releaseID), filepath.FromSlash(strings.TrimPrefix(relPath, "/")))
}

func ensureSiteWorkspace(root, tenantID, siteID string) error {
	return os.MkdirAll(siteWorkspaceDir(root, tenantID, siteID), 0o755)
}

func ensureSitePublished(root, tenantID, siteID string) error {
	return os.MkdirAll(sitePublishedDir(root, tenantID, siteID), 0o755)
}

func ensureSiteReleases(root, tenantID, siteID string) error {
	return os.MkdirAll(siteReleasesDir(root, tenantID, siteID), 0o755)
}

func cleanPublicSlug(slug string) string {
	slug = strings.TrimSpace(strings.ToLower(slug))
	slug = strings.ReplaceAll(slug, " ", "-")
	return slug
}
