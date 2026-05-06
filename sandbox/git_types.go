package sandbox

// GitFileStatus 描述单个文件的 git 状态条目。
type GitFileStatus struct {
	// Name 是相对于仓库根目录的路径。
	Name string
	// Status 是归一化后的状态字符串，例如 "modified"、"added"、"deleted"、"untracked"、"conflict" 等。
	Status string
	// IndexStatus 是 porcelain 输出中索引位的字符。
	IndexStatus string
	// WorkingTreeStatus 是 porcelain 输出中工作区位的字符。
	WorkingTreeStatus string
	// Staged 表示该文件是否已暂存。
	Staged bool
	// RenamedFrom 在文件被重命名时记录原路径。
	RenamedFrom string
}

// GitStatus 描述仓库整体状态。
type GitStatus struct {
	// CurrentBranch 是当前分支名，HEAD 分离时为空。
	CurrentBranch string
	// Upstream 是上游分支名，未配置时为空。
	Upstream string
	// Ahead 是当前分支领先上游的提交数。
	Ahead int
	// Behind 是当前分支落后上游的提交数。
	Behind int
	// Detached 表示 HEAD 是否处于分离状态。
	Detached bool
	// FileStatus 是所有有变更的文件状态条目。
	FileStatus []GitFileStatus
}

// IsClean 在仓库无任何文件变更时返回 true。
func (s *GitStatus) IsClean() bool {
	return len(s.FileStatus) == 0
}

// HasChanges 在仓库存在任意文件变更时返回 true。
func (s *GitStatus) HasChanges() bool {
	return len(s.FileStatus) > 0
}

// HasStaged 在至少一个文件存在已暂存变更时返回 true。
func (s *GitStatus) HasStaged() bool {
	for i := range s.FileStatus {
		if s.FileStatus[i].Staged {
			return true
		}
	}
	return false
}

// HasUntracked 在存在未跟踪文件时返回 true。
func (s *GitStatus) HasUntracked() bool {
	for i := range s.FileStatus {
		if s.FileStatus[i].Status == "untracked" {
			return true
		}
	}
	return false
}

// HasConflicts 在存在冲突文件时返回 true。
func (s *GitStatus) HasConflicts() bool {
	for i := range s.FileStatus {
		if s.FileStatus[i].Status == "conflict" {
			return true
		}
	}
	return false
}

// TotalCount 返回有变更的文件总数。
func (s *GitStatus) TotalCount() int {
	return len(s.FileStatus)
}

// StagedCount 返回已暂存的文件数。
func (s *GitStatus) StagedCount() int {
	n := 0
	for i := range s.FileStatus {
		if s.FileStatus[i].Staged {
			n++
		}
	}
	return n
}

// UnstagedCount 返回未暂存的文件数。
func (s *GitStatus) UnstagedCount() int {
	return len(s.FileStatus) - s.StagedCount()
}

// UntrackedCount 返回未跟踪的文件数。
func (s *GitStatus) UntrackedCount() int {
	n := 0
	for i := range s.FileStatus {
		if s.FileStatus[i].Status == "untracked" {
			n++
		}
	}
	return n
}

// ConflictCount 返回冲突文件数。
func (s *GitStatus) ConflictCount() int {
	n := 0
	for i := range s.FileStatus {
		if s.FileStatus[i].Status == "conflict" {
			n++
		}
	}
	return n
}

// GitBranches 描述仓库的分支列表。
type GitBranches struct {
	// Branches 是所有本地分支名。
	Branches []string
	// CurrentBranch 是当前分支名，HEAD 分离时为空。
	CurrentBranch string
}

// GitConfigScope 描述 git config 命令的作用域。
type GitConfigScope string

const (
	// GitConfigScopeGlobal 表示用户级配置（~/.gitconfig）。
	GitConfigScopeGlobal GitConfigScope = "global"
	// GitConfigScopeLocal 表示仓库级配置（<repo>/.git/config），需要同时提供仓库路径。
	GitConfigScopeLocal GitConfigScope = "local"
	// GitConfigScopeSystem 表示系统级配置（/etc/gitconfig）。
	GitConfigScopeSystem GitConfigScope = "system"
)
