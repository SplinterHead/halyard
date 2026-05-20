package manager

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/SplinterHead/halyard/api"
)

type GitManager struct {
	db *DB
	mu sync.RWMutex
}

func NewGitManager(db *DB) *GitManager {
	return &GitManager{
		db: db,
	}
}

func (m *GitManager) ListRepos() []api.GitRepository {
	rows, err := m.db.Query("SELECT id, name, url, description, username, token, last_status, last_error, created_at FROM git_sources")
	if err != nil {
		return []api.GitRepository{}
	}
	defer rows.Close()

	list := make([]api.GitRepository, 0)
	for rows.Next() {
		var r api.GitRepository
		rows.Scan(&r.ID, &r.Name, &r.URL, &r.Description, &r.Username, &r.Token, &r.LastStatus, &r.LastError, &r.CreatedAt)
		list = append(list, r)
	}
	return list
}

func (m *GitManager) AddRepo(repo api.GitRepository) (api.GitRepository, error) {
	err := m.TestConnection(repo)
	if err != nil {
		repo.LastStatus = "Failed"
		repo.LastError = err.Error()
	} else {
		repo.LastStatus = "Success"
		repo.LastError = ""
	}

	b := make([]byte, 16)
	rand.Read(b)
	repo.ID = hex.EncodeToString(b)
	repo.CreatedAt = time.Now()

	_, err = m.db.Exec(`INSERT INTO git_sources (id, name, url, description, username, token, last_status, last_error, created_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		repo.ID, repo.Name, repo.URL, repo.Description, repo.Username, repo.Token, repo.LastStatus, repo.LastError, repo.CreatedAt)

	return repo, err
}

func (m *GitManager) GetRepo(id string) (api.GitRepository, bool) {
	var r api.GitRepository
	err := m.db.QueryRow("SELECT id, name, url, description, username, token, last_status, last_error, created_at FROM git_sources WHERE id = ?", id).
		Scan(&r.ID, &r.Name, &r.URL, &r.Description, &r.Username, &r.Token, &r.LastStatus, &r.LastError, &r.CreatedAt)

	return r, err == nil
}

func (m *GitManager) UpdateRepo(repo api.GitRepository) (api.GitRepository, error) {
	err := m.TestConnection(repo)
	if err != nil {
		repo.LastStatus = "Failed"
		repo.LastError = err.Error()
	} else {
		repo.LastStatus = "Success"
		repo.LastError = ""
	}

	_, err = m.db.Exec(`UPDATE git_sources SET name = ?, url = ?, description = ?, username = ?, token = ?, last_status = ?, last_error = ? WHERE id = ?`,
		repo.Name, repo.URL, repo.Description, repo.Username, repo.Token, repo.LastStatus, repo.LastError, repo.ID)

	return repo, err
}

func (m *GitManager) TestRepo(id string) error {
	repo, ok := m.GetRepo(id)
	if !ok {
		return fmt.Errorf("repo not found")
	}

	err := m.TestConnection(repo)
	status := "Success"
	errMsg := ""
	if err != nil {
		status = "Failed"
		errMsg = err.Error()
	}

	_, dbErr := m.db.Exec("UPDATE git_sources SET last_status = ?, last_error = ? WHERE id = ?", status, errMsg, id)
	if dbErr != nil {
		return dbErr
	}
	return err
}

func (m *GitManager) DeleteRepo(id string) error {
	// Also delete associated syncs to maintain integrity
	_, err := m.db.Exec("DELETE FROM git_syncs WHERE source_id = ?", id)
	if err != nil {
		return err
	}
	_, err = m.db.Exec("DELETE FROM git_sources WHERE id = ?", id)
	return err
}

func (m *GitManager) ListFiles(repo api.GitRepository, branch, rootPath string) ([]api.FileEntry, error) {
	auth := &http.BasicAuth{
		Username: repo.Username,
		Password: repo.Token,
	}

	r, err := git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:           repo.URL,
		Auth:          auth,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		return nil, err
	}

	head, err := r.Head()
	if err != nil {
		return nil, err
	}

	commit, err := r.CommitObject(head.Hash())
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	if rootPath != "" && rootPath != "." {
		tree, err = tree.Tree(rootPath)
		if err != nil {
			return nil, err
		}
	}

	entries := make([]api.FileEntry, 0)
	for _, entry := range tree.Entries {
		entries = append(entries, api.FileEntry{
			Name:  entry.Name,
			Path:  path.Join(rootPath, entry.Name),
			IsDir: entry.Mode.IsFile() == false,
		})
	}

	return entries, nil
}

func (m *GitManager) ListBranches(repo api.GitRepository) ([]string, error) {
	auth := &http.BasicAuth{
		Username: repo.Username,
		Password: repo.Token,
	}

	rem := git.NewRemote(nil, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repo.URL},
	})

	refs, err := rem.List(&git.ListOptions{
		Auth: auth,
	})
	if err != nil {
		return nil, err
	}

	branches := make([]string, 0)
	for _, ref := range refs {
		if ref.Name().IsBranch() {
			branches = append(branches, ref.Name().Short())
		}
	}
	return branches, nil
}

func (m *GitManager) CheckFile(repo api.GitRepository, branch, path string) error {
	auth := &http.BasicAuth{
		Username: repo.Username,
		Password: repo.Token,
	}

	r, err := git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:           repo.URL,
		Auth:          auth,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		return err
	}

	head, err := r.Head()
	if err != nil {
		return err
	}

	commit, err := r.CommitObject(head.Hash())
	if err != nil {
		return err
	}

	tree, err := commit.Tree()
	if err != nil {
		return err
	}

	_, err = tree.File(path)
	return err
}

func (m *GitManager) CloneToDisk(repo api.GitRepository, branch string) (string, func(), error) {
	auth := &http.BasicAuth{
		Username: repo.Username,
		Password: repo.Token,
	}

	tmpDir, err := os.MkdirTemp("", "halyard-git-*")
	if err != nil {
		return "", nil, err
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	_, err = git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL:           repo.URL,
		Auth:          auth,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		cleanup()
		return "", nil, err
	}

	return tmpDir, cleanup, nil
}

func (m *GitManager) GetFileContent(repo api.GitRepository, branch, path string) (string, error) {
	auth := &http.BasicAuth{
		Username: repo.Username,
		Password: repo.Token,
	}

	r, err := git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:           repo.URL,
		Auth:          auth,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		return "", err
	}

	head, err := r.Head()
	if err != nil {
		return "", err
	}

	commit, err := r.CommitObject(head.Hash())
	if err != nil {
		return "", err
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", err
	}

	file, err := tree.File(path)
	if err != nil {
		return "", err
	}

	return file.Contents()
}

func (m *GitManager) GetLatestSHA(repo api.GitRepository, branch string) (string, error) {
	auth := &http.BasicAuth{
		Username: repo.Username,
		Password: repo.Token,
	}

	rem := git.NewRemote(nil, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repo.URL},
	})

	refs, err := rem.List(&git.ListOptions{
		Auth: auth,
	})
	if err != nil {
		return "", err
	}

	for _, ref := range refs {
		if ref.Name().Short() == branch {
			return ref.Hash().String(), nil
		}
	}

	return "", fmt.Errorf("branch %s not found", branch)
}

func (m *GitManager) TestConnection(repo api.GitRepository) error {
	if repo.URL == "" {
		return errors.New("repository URL is required")
	}

	auth := &http.BasicAuth{
		Username: repo.Username,
		Password: repo.Token,
	}

	rem := git.NewRemote(nil, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repo.URL},
	})

	_, err := rem.List(&git.ListOptions{
		Auth: auth,
	})

	return err
}
