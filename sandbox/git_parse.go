package sandbox

import (
	"net/url"
	"path"
	"strconv"
	"strings"
)

// deriveRepoDirFromURL 从 git URL 推导默认的仓库目录名（去除 .git 后缀）。
func deriveRepoDirFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	var name string
	if err == nil && u.Path != "" {
		name = path.Base(u.Path)
	} else {
		// 处理形如 git@host:owner/repo.git 的 SCP 风格地址。
		if idx := strings.LastIndex(rawURL, ":"); idx >= 0 {
			name = path.Base(rawURL[idx+1:])
		} else {
			name = path.Base(rawURL)
		}
	}
	name = strings.TrimSuffix(name, ".git")
	if name == "." || name == "/" || name == "" {
		return ""
	}
	return name
}

// parseGitStatus 解析 `git status --porcelain=1 -b` 的输出。
func parseGitStatus(out string) *GitStatus {
	status := &GitStatus{}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			parseBranchLine(line[3:], status)
			continue
		}
		entry, ok := parseFileStatusLine(line)
		if ok {
			status.FileStatus = append(status.FileStatus, entry)
		}
	}
	return status
}

// parseBranchLine 解析 porcelain 输出首行的分支信息。
//
// 需处理 git 在多种状态下的输出形态：
//   - "main...origin/main [ahead 1, behind 2]" 普通跟踪分支
//   - "HEAD (no branch)"                       传统 detached HEAD（如 rebase 中）
//   - "HEAD (detached at <ref>)"               显式 detached HEAD
//   - "No commits yet on main"                 unborn 分支（仓库未首次提交）
//   - "Initial commit on main"                 旧版 git 的 unborn 写法
//
// 不能简单按第一个空格切分，否则 "No commits yet on main" 会把 "No" 当成分支名。
func parseBranchLine(line string, status *GitStatus) {
	if line == "" {
		return
	}

	// 先剥离尾部的 "[ahead N, behind M]"
	branchPart := line
	rest := ""
	if i := strings.Index(line, " ["); i >= 0 && strings.HasSuffix(line, "]") {
		branchPart = line[:i]
		rest = line[i+2 : len(line)-1]
	}

	// 处理 detached HEAD 的两种形态。
	if strings.HasPrefix(branchPart, "HEAD (no branch)") ||
		strings.HasPrefix(branchPart, "HEAD (detached") ||
		strings.HasPrefix(branchPart, "(no branch)") {
		status.Detached = true
		parseAheadBehind(rest, status)
		return
	}

	// 处理 unborn 分支：尚未首次提交时 git 会输出 "No commits yet on <branch>"
	// 或更早版本的 "Initial commit on <branch>"。此时仍记录分支名，但没有 upstream。
	switch {
	case strings.HasPrefix(branchPart, "No commits yet on "):
		status.CurrentBranch = strings.TrimPrefix(branchPart, "No commits yet on ")
	case strings.HasPrefix(branchPart, "Initial commit on "):
		status.CurrentBranch = strings.TrimPrefix(branchPart, "Initial commit on ")
	default:
		if before, after, ok := strings.Cut(branchPart, "..."); ok {
			status.CurrentBranch = before
			status.Upstream = after
		} else {
			status.CurrentBranch = branchPart
		}
	}

	parseAheadBehind(rest, status)
}

// parseAheadBehind 解析 ahead/behind 段（不含外层方括号）。
func parseAheadBehind(rest string, status *GitStatus) {
	if rest == "" {
		return
	}
	for _, seg := range strings.Split(rest, ",") {
		seg = strings.TrimSpace(seg)
		switch {
		case strings.HasPrefix(seg, "ahead "):
			if n, err := strconv.Atoi(strings.TrimPrefix(seg, "ahead ")); err == nil {
				status.Ahead = n
			}
		case strings.HasPrefix(seg, "behind "):
			if n, err := strconv.Atoi(strings.TrimPrefix(seg, "behind ")); err == nil {
				status.Behind = n
			}
		}
	}
}

// parseFileStatusLine 解析 porcelain 输出的单条文件状态行。
//
// 当文件名包含空格、双引号或非可打印字符时，git 默认会输出 C-style 引号转义
// 的路径（如 `?? "with space.txt"`、`R  "old name" -> "new name"`），这里需要
// 对引号内容做反转义，确保调用方拿到的是真正的相对路径。
func parseFileStatusLine(line string) (GitFileStatus, bool) {
	if len(line) < 3 {
		return GitFileStatus{}, false
	}
	x := string(line[0])
	y := string(line[1])
	rest := line[3:]

	entry := GitFileStatus{
		IndexStatus:       x,
		WorkingTreeStatus: y,
	}

	// 重命名/复制：路径形如 "old -> new"，两端可能各自被引号包裹。
	if x == "R" || x == "C" || y == "R" || y == "C" {
		if before, after, ok := splitRenamePath(rest); ok {
			entry.RenamedFrom = unquoteCPath(before)
			entry.Name = unquoteCPath(after)
		} else {
			entry.Name = unquoteCPath(rest)
		}
	} else {
		entry.Name = unquoteCPath(rest)
	}

	entry.Status = normalizeFileStatus(x, y)
	entry.Staged = isStaged(x, entry.Status)
	return entry, true
}

// splitRenamePath 拆分 "old -> new" 形式的重命名路径，正确处理两端被引号包裹的情况。
// 普通情况下直接按 " -> " 切分；若 old 部分以 `"` 开头，则定位到匹配的结束引号后再找 " -> "。
func splitRenamePath(s string) (string, string, bool) {
	if strings.HasPrefix(s, `"`) {
		// 找到第一个未被反斜杠转义的结束引号。
		i := 1
		for i < len(s) {
			if s[i] == '\\' && i+1 < len(s) {
				i += 2
				continue
			}
			if s[i] == '"' {
				break
			}
			i++
		}
		if i >= len(s) {
			return "", "", false
		}
		head := s[:i+1]
		rest := s[i+1:]
		const sep = " -> "
		if strings.HasPrefix(rest, sep) {
			return head, rest[len(sep):], true
		}
		return "", "", false
	}
	before, after, ok := strings.Cut(s, " -> ")
	return before, after, ok
}

// unquoteCPath 对 git porcelain 输出的 C-style 引号路径做反转义。
//
// 输入未以双引号开头时原样返回；否则按 git 的 quote_c_style 规则解析：
// 支持 \\、\"、\a \b \f \n \r \t \v 以及 \<3-octal> 任意字节。
// 解析失败时回退到原字符串，避免抛错破坏调用流程。
func unquoteCPath(s string) string {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return s
	}
	body := s[1 : len(s)-1]
	out := make([]byte, 0, len(body))
	for i := 0; i < len(body); i++ {
		c := body[i]
		if c != '\\' {
			out = append(out, c)
			continue
		}
		if i+1 >= len(body) {
			return s
		}
		next := body[i+1]
		switch next {
		case '\\', '"', '\'':
			out = append(out, next)
			i++
		case 'a':
			out = append(out, '\a')
			i++
		case 'b':
			out = append(out, '\b')
			i++
		case 'f':
			out = append(out, '\f')
			i++
		case 'n':
			out = append(out, '\n')
			i++
		case 'r':
			out = append(out, '\r')
			i++
		case 't':
			out = append(out, '\t')
			i++
		case 'v':
			out = append(out, '\v')
			i++
		default:
			// 三位八进制：\NNN
			if i+3 < len(body) && isOctal(body[i+1]) && isOctal(body[i+2]) && isOctal(body[i+3]) {
				v := (int(body[i+1]-'0') << 6) | (int(body[i+2]-'0') << 3) | int(body[i+3]-'0')
				out = append(out, byte(v))
				i += 3
				continue
			}
			return s
		}
	}
	return string(out)
}

// isOctal 判断字节是否为八进制数字字符。
func isOctal(c byte) bool { return c >= '0' && c <= '7' }

// normalizeFileStatus 将 XY 字符归一化为可读状态。
func normalizeFileStatus(x, y string) string {
	if x == "?" && y == "?" {
		return "untracked"
	}
	if x == "!" && y == "!" {
		return "ignored"
	}
	if isConflict(x, y) {
		return "conflict"
	}
	// 优先看索引位，其次工作区位
	if s := mapStatusChar(x); s != "" && s != "unmodified" {
		return s
	}
	if s := mapStatusChar(y); s != "" && s != "unmodified" {
		return s
	}
	return "unmodified"
}

// mapStatusChar 将单个状态字符映射为可读字符串。
func mapStatusChar(c string) string {
	switch c {
	case " ":
		return "unmodified"
	case "M":
		return "modified"
	case "A":
		return "added"
	case "D":
		return "deleted"
	case "R":
		return "renamed"
	case "C":
		return "copied"
	case "T":
		return "type-changed"
	case "U":
		return "conflict"
	}
	return ""
}

// isConflict 判断 XY 是否构成 git 合并冲突。
// 参考 git status 文档中 "Unmerged" 表（DD/AU/UD/UA/DU/AA/UU）。
func isConflict(x, y string) bool {
	switch x + y {
	case "DD", "AU", "UD", "UA", "DU", "AA", "UU":
		return true
	}
	return x == "U" || y == "U"
}

// isStaged 判断条目是否处于已暂存状态。
func isStaged(x, status string) bool {
	if status == "untracked" || status == "ignored" || status == "conflict" {
		return false
	}
	if x == " " || x == "?" || x == "!" {
		return false
	}
	return true
}

// parseGitBranches 解析 `git branch --format=%(refname:short)\t%(HEAD)` 的输出。
//
// detached HEAD 时 git 会把 "(HEAD detached at <sha>)" / "(HEAD detached from <ref>)"
// 作为一项输出并标记为当前 HEAD；这里跳过该伪分支以保持 CurrentBranch 在 HEAD 分离时为空。
func parseGitBranches(out string) *GitBranches {
	result := &GitBranches{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}
		isCurrent := len(parts) == 2 && strings.TrimSpace(parts[1]) == "*"
		if strings.HasPrefix(name, "(HEAD detached") || name == "(no branch)" {
			continue
		}
		result.Branches = append(result.Branches, name)
		if isCurrent {
			result.CurrentBranch = name
		}
	}
	return result
}
