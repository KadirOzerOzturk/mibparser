package mibparser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

type OIDNode struct {
	Name        string     `json:"name"`
	OID         string     `json:"oid"`
	ID          string     `json:"id"`
	Parent      string     `json:"parent"`
	Description string     `json:"description"`
	Children    []*OIDNode `json:"children,omitempty"`
}

func (p *MIBParser) Parse() ([]OIDNode, error) {
	lines, err := p.ReadMIBFile()
	if err != nil {
		return nil, err
	}
	nodes, err := parseMIB(lines)
	if err != nil {
		log.Fatalf(err.Error())
	}
	return nodes, err
}

func (p *MIBParser) GetJSONTree() (string, error) {
	nodes, err := p.Parse()
	if err != nil {
		return "", err
	}
	if err := saveNodesToJSON(nodes, p.opts.Path+".json"); err != nil {
		return "", err
	}
	tree, err := buildTree(nodes)
	if err != nil {
		return "", err
	}
	jsonTree, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return "", err
	}
	if err := saveTreeToJSON(jsonTree, p.opts.Path+"-tree.json"); err != nil {
		return "", err
	}
	return string(jsonTree), nil
}

func saveNodesToJSON(nodes []OIDNode, filename string) error {
	jsonData, err := json.MarshalIndent(nodes, "", "  ")
	if err != nil {
		return fmt.Errorf("error converting nodes to JSON: %w", err)
	}
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating JSON file: %w", err)
	}
	defer file.Close()
	_, err = file.Write(jsonData)
	if err != nil {
		return fmt.Errorf("error writing JSON data to file: %w", err)
	}
	return nil
}

func buildTree(nodes []OIDNode) ([]*OIDNode, error) {
	nodeMap := make(map[string]*OIDNode)
	// Initialize the nodeMap
	for i := range nodes {
		node := &nodes[i]
		nodeMap[node.Name] = node
	}

	var rootNodes []*OIDNode
	rootNames := map[string]bool{"iso": true}
	for i := range nodes {
		node := &nodes[i]
		if rootNames[node.Parent] {
			rootNodes = append(rootNodes, node)
		} else if parent, exists := nodeMap[node.Parent]; exists {
			parent.Children = append(parent.Children, node)
		}
	}
	return rootNodes, nil
}

func saveTreeToJSON(jsonTree []byte, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating JSON file: %w", err)
	}
	defer file.Close()
	_, err = file.Write(jsonTree)
	if err != nil {
		return fmt.Errorf("error writing JSON data to file: %w", err)
	}
	return nil
}
func parseMIB(lines []string) ([]OIDNode, error) {
	var requiredMibs []string
	var definingMibs []string
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.Contains(line, "FROM") {
			parts := strings.Split(line, "FROM")
			requiredMib := strings.TrimSpace(strings.Split(parts[1], ";")[0])

			requiredMibs = append(requiredMibs, requiredMib+".mib")

		}
		if strings.Contains(line, "DEFINITIONS ::= BEGIN") {

			definingMib := strings.TrimSpace(strings.Split(line, "DEFINITIONS ::= BEGIN")[0])
			definingMibs = append(definingMibs, definingMib+".mib")
		}
	}
	for i := len(requiredMibs) - 1; i >= 0; i-- {
		for _, definingMib := range definingMibs {
			if definingMib == requiredMibs[i] {
				requiredMibs = append(requiredMibs[:i], requiredMibs[i+1:]...)
				break
			}
		}
	}

	uniqueMap := make(map[string]bool)
	uniqueReqireds := []string{}

	for _, item := range requiredMibs {
		if _, found := uniqueMap[item]; !found {
			uniqueMap[item] = true
			uniqueReqireds = append(uniqueReqireds, item)
		}
	}

	if len(uniqueReqireds) > 0 {
		return nil, fmt.Errorf("parsing operation has failed :\n \t\t\t\t\trequired files: %s", uniqueReqireds)
	}

	var nodes []OIDNode

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.Contains(line, "OBJECT IDENTIFIER ::= {") {
			parts := strings.Split(line, "OBJECT IDENTIFIER ::= {")
			name := strings.TrimSpace(strings.Split(parts[0], " ")[0])
			nextNode := strings.TrimSpace(strings.Trim(parts[1], "}"))

			nodes = append(nodes, OIDNode{
				Name:        name,
				ID:          strings.Split(nextNode, " ")[1],
				Parent:      strings.Split(nextNode, " ")[0],
				Description: "",
			})
		} else if (strings.Contains(line, "OBJECT-TYPE") || strings.Contains(line, "OBJECT-IDENTITY")) && !strings.Contains(line, "MODULE-IDENTITY") {
			name := strings.TrimSpace(strings.Fields(line)[0])
			parent := ""
			description := ""
			for j := i + 1; j < len(lines); j++ {
				nextLine := strings.TrimSpace(lines[j])
				if strings.HasPrefix(nextLine, "::= {") {
					parent = strings.Trim(strings.Split(nextLine, "{")[1], " }")
					break
				}
				if strings.HasPrefix(nextLine, "DESCRIPTION") {
					descriptionLine := nextLine
					for k := j + 1; k < len(lines); k++ {
						if strings.Contains(lines[k], "::= {") {
							break
						}
						descriptionLine += " " + strings.TrimSpace(lines[k])
					}
					description = strings.TrimSpace(strings.Trim(descriptionLine, "DESCRIPTION"))
				}
			}
			if name != "OBJECT-TYPE" && name != "--" {
				nodes = append(nodes, OIDNode{
					Name:        name,
					ID:          strings.Split(parent, " ")[1],
					Parent:      strings.Split(parent, " ")[0],
					Description: description,
				})
			}

		}

	}
	formatedNodes, err := setOids(nodes)
	if err != nil {
		log.Fatalf(err.Error())
	}
	return formatedNodes, nil
}

func setOids(nodes []OIDNode) ([]OIDNode, error) {

	var formatedNodes []OIDNode
	nodeMap := make(map[string]OIDNode)
	for _, node := range nodes {
		nodeMap[node.Name] = node
	}
	for _, node := range nodes {
		oidParts := []string{node.ID}
		parent := node.Parent

		for parent != "" {
			if nextNode, found := nodeMap[parent]; found {
				oidParts = append([]string{nextNode.ID}, oidParts...)
				parent = nextNode.Parent

			} else {
				break
			}
		}

		oid := strings.Join(oidParts, ".")
		formatedNodes = append(formatedNodes, OIDNode{
			Name:        node.Name,
			ID:          node.ID,
			Parent:      node.Parent,
			OID:         "1." + oid,
			Description: node.Description,
		})
	}

	return formatedNodes, nil
}
