package server

import (
	"regexp"
	"sort"
)

type APIResponse struct {
	Header struct {
		IsSuccessful  bool   `json:"isSuccessful"`
		ResultCode    int    `json:"resultCode"`
		ResultMessage string `json:"resultMessage"`
	} `json:"header"`
	Body struct {
		TotalCount int       `json:"totalCount"`
		Data       []ESMNode `json:"data"`
	} `json:"body"`
}

type ESMNode struct {
	ESMName      string     `json:"esmName"`
	ESMCode      string     `json:"esmCode"`
	ESMFullName  string     `json:"esmFullName"`
	CIServerList []CIServer `json:"ciServerList"`
}

type CIServer struct {
	HostName        string        `json:"hostName"`
	ServerState     string        `json:"serverState"`
	ManagerName     string        `json:"managerName"`
	DeveloperName   string        `json:"developerName"`
	IP              string        `json:"ip"`
	IDCName         string        `json:"idcName"`
	SpecCode        string        `json:"specCode"`
	OSName          string        `json:"osName"`
	ServerGroupList []ServerGroup `json:"serverGroupList"`
	ESMList         []ESMRef      `json:"esmList"`
}

type ServerGroup struct {
	ServerGroupName   string   `json:"serverGroupName"`
	ServerUsageName   string   `json:"serverUsageName"`
	ServerUseTypeName string   `json:"serverUseTypeName"`
	ServerGroupRemark string   `json:"serverGroupRemark"`
	ServerTags        []string `json:"serverTags"`
	ServerRemark      string   `json:"serverRemark"`
}

type ESMRef struct {
	ESMName     string `json:"esmName"`
	ESMFullName string `json:"esmFullName"`
	ESMCode     string `json:"esmCode"`
}

// ViewMode determines how servers are grouped in the TUI.
type ViewMode int

const (
	ViewByESM         ViewMode = iota // 1: ESM별
	ViewByServerGroup                 // 2: 서버그룹별
	ViewByPrefix                      // 3: Prefix별
)

func (v ViewMode) Label() string {
	switch v {
	case ViewByESM:
		return "ESM"
	case ViewByServerGroup:
		return "ServerGroup"
	case ViewByPrefix:
		return "Prefix"
	}
	return ""
}

func (v ViewMode) Next() ViewMode {
	return (v + 1) % 3
}

// GroupView is a unified group for display, regardless of ViewMode.
type GroupView struct {
	GroupName string
	Badge     string // e.g. "[APP|서비스]" or "[NE032765]"
	Remark    string
	Servers   []CIServer
}

// BuildGroups creates groups from ESM nodes based on the given ViewMode.
func BuildGroups(nodes []ESMNode, mode ViewMode) []GroupView {
	switch mode {
	case ViewByESM:
		return groupByESM(nodes)
	case ViewByServerGroup:
		return groupByServerGroup(nodes)
	case ViewByPrefix:
		return groupByPrefix(nodes)
	}
	return nil
}

func groupByESM(nodes []ESMNode) []GroupView {
	var groups []GroupView
	for _, node := range nodes {
		if len(node.CIServerList) == 0 {
			continue
		}
		groups = append(groups, GroupView{
			GroupName: node.ESMName,
			Badge:     node.ESMCode,
			Servers:   node.CIServerList,
		})
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].GroupName < groups[j].GroupName
	})
	return groups
}

func groupByServerGroup(nodes []ESMNode) []GroupView {
	groupMap := make(map[string]*GroupView)
	var order []string

	for _, node := range nodes {
		for _, srv := range node.CIServerList {
			if len(srv.ServerGroupList) == 0 {
				key := "(ungrouped)"
				if _, ok := groupMap[key]; !ok {
					groupMap[key] = &GroupView{GroupName: key}
					order = append(order, key)
				}
				groupMap[key].Servers = append(groupMap[key].Servers, srv)
				continue
			}
			for _, sg := range srv.ServerGroupList {
				// Composite key: name + usage + type
				key := sg.ServerGroupName + "\x00" + sg.ServerUsageName + "\x00" + sg.ServerUseTypeName
				if _, ok := groupMap[key]; !ok {
					badge := ""
					if sg.ServerUseTypeName != "" || sg.ServerUsageName != "" {
						var parts []string
						if sg.ServerUseTypeName != "" {
							parts = append(parts, sg.ServerUseTypeName)
						}
						if sg.ServerUsageName != "" {
							parts = append(parts, sg.ServerUsageName)
						}
						badge = joinParts(parts, "|")
					}
					groupMap[key] = &GroupView{
						GroupName: sg.ServerGroupName,
						Badge:     badge,
						Remark:    sg.ServerGroupRemark,
					}
					order = append(order, key)
				}
				groupMap[key].Servers = append(groupMap[key].Servers, srv)
			}
		}
	}

	sort.Strings(order)
	result := make([]GroupView, 0, len(order))
	for _, key := range order {
		result = append(result, *groupMap[key])
	}
	return result
}

// prefixRe strips trailing -gNNNN, -cNNNN, -tNNNN, -pNNNN or just trailing digits
var prefixRe = regexp.MustCompile(`[-_]([gctpGCTP]?\d{3,})$`)

func hostnamePrefix(hostname string) string {
	return prefixRe.ReplaceAllString(hostname, "")
}

func groupByPrefix(nodes []ESMNode) []GroupView {
	groupMap := make(map[string]*GroupView)
	var order []string

	for _, node := range nodes {
		for _, srv := range node.CIServerList {
			prefix := hostnamePrefix(srv.HostName)
			if _, ok := groupMap[prefix]; !ok {
				groupMap[prefix] = &GroupView{GroupName: prefix}
				order = append(order, prefix)
			}
			groupMap[prefix].Servers = append(groupMap[prefix].Servers, srv)
		}
	}

	sort.Strings(order)
	result := make([]GroupView, 0, len(order))
	for _, key := range order {
		result = append(result, *groupMap[key])
	}
	return result
}

func joinParts(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// PrimaryIP returns the first IP address (before any semicolons).
func (s *CIServer) PrimaryIP() string {
	ip := s.IP
	if idx := indexOf(ip, ';'); idx >= 0 {
		ip = ip[:idx]
	}
	return trimSpace(ip)
}

// AllTags collects all tags from all server groups.
func (s *CIServer) AllTags() []string {
	seen := make(map[string]bool)
	var tags []string
	for _, sg := range s.ServerGroupList {
		for _, t := range sg.ServerTags {
			if !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}
	return tags
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && s[start] == ' ' {
		start++
	}
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}
