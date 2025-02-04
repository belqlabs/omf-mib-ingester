package main

import (
	"crypto/md5"
	"fmt"
	"hash/adler32"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/belqlabs/omf-gosmi"
	"github.com/belqlabs/omf-gosmi/models"
	gosmi_types "github.com/belqlabs/omf-gosmi/types"
)

type OMFEnum = map[string]int64

type OMFRange = [2]int64

type OMFType struct {
	BaseType    string     `json:",omitempty"`
	Decl        string     `json:",omitempty"`
	Description string     `json:",omitempty"`
	Enum        OMFEnum    `json:",omitempty"`
	Format      string     `json:",omitempty"`
	Name        string     `json:",omitempty"`
	Ranges      []OMFRange `json:",omitempty"`
	Reference   string     `json:",omitempty"`
	Status      string     `json:",omitempty"`
	Units       string     `json:",omitempty"`
}

type OMFNode struct {
	NodeHash    string   `json:",omitempty"`
	Access      string   `json:",omitempty"`
	Decl        string   `json:",omitempty"`
	Kind        string   `json:",omitempty"`
	Name        string   `json:",omitempty"`
	Module      string   `json:",omitempty"`
	Description string   `json:",omitempty"`
	Oid         string   `json:",omitempty"`
	Status      string   `json:",omitempty"`
	Type        *OMFType `json:",omitempty"`
}

type OMFTable struct {
	OMFNode
	Entry   OMFNode
	Indexes []OMFIndex
	Columns []OMFNode
}

type OMFIndex struct {
	OMFNode
}

type OMFNotification struct {
	OMFNode
	Objects []OMFNode
}

type OMFScalar struct {
	OMFNode
}

type OMFImport struct {
	ModName       string
	ImportedNodes []string
}

type OMFRevision struct {
	Date        time.Time
	Description string
}

type OMFTextualConvention struct {
	OMFType
}

type OMFModule struct {
	ModuleHash         string
	ContactInfo        string
	Description        string
	Language           string
	Name               string
	Organization       string
	Path               string
	Reference          string
	Imports            []OMFImport
	Revisions          []OMFRevision
	Scalars            []OMFScalar
	Indexes            []OMFIndex
	Types              []OMFType
	TextualConventions []OMFTextualConvention
	Tables             []OMFTable
	Notifications      []OMFNotification
	OtherNodes         []OMFNode
}

type OMFTreeNode struct {
	NodeOid  string
	Node     OMFNode
	Children map[string]*OMFTreeNode `json:",omitempty"`
}

type OMFTree struct {
	ModuleHash    string
	ContactInfo   string
	Description   string
	Language      string
	Name          string
	Organization  string
	Path          string
	Reference     string
	ModuleImports *[]OMFImport `json:",omitempty"`
	RootNode      *OMFTreeNode
}

type OMFTypeConstraint interface {
	OMFModule | OMFTextualConvention | OMFRevision | OMFImport | OMFScalar | OMFNotification | OMFIndex | OMFTable | OMFNode | OMFType | OMFRange | OMFEnum
}

func join_imports(imps []models.Import) map[string][]string {
	import_map := make(map[string][]string)

	for _, ip := range imps {

		if len(import_map[ip.Module]) == 0 {
			import_map[ip.Module] = []string{ip.Name}
			continue
		}

		import_map[ip.Module] = append(import_map[ip.Module], ip.Name)

	}

	return import_map
}

func init_gosmi(path string, module_name string) {
	gosmi.Init()

	gosmi.AppendPath(path)

	_, err := gosmi.LoadModule(module_name)
	if err != nil {
		fmt.Printf("Init Error: %s\n", err)
		return
	}
}

func exit_gosmi() {
	gosmi.Exit()
}

func omfy_enum(en *models.Enum) OMFEnum {
	omf_enum := make(OMFEnum)

	if en == nil {
		return omf_enum
	}

	for _, v := range en.Values {
		omf_enum[v.Name] = v.Value
	}

	return omf_enum
}

func omfy_ranges(rgs []models.Range) []OMFRange {
	var omf_ranges []OMFRange

	if rgs == nil {
		return omf_ranges
	}

	for _, rg := range rgs {
		new_range := OMFRange{rg.MinValue, rg.MaxValue}

		omf_ranges = append(omf_ranges, new_range)
	}

	return omf_ranges
}

func omfy_type(tp *models.Type) OMFType {
	if tp == nil {
		return OMFType{}
	}

	return OMFType{
		BaseType:    tp.BaseType.String(),
		Decl:        tp.Decl.String(),
		Description: tp.Description,
		Enum:        omfy_enum(tp.Enum),
		Format:      tp.Format,
		Name:        tp.Name,
		Ranges:      omfy_ranges(tp.Ranges),
		Reference:   tp.Reference,
		Status:      tp.Status.String(),
		Units:       tp.Units,
	}
}

func generate_node_hash(nd *OMFNode) string {

	mod_hash_string := fmt.Sprintf("%s%s%s%s", nd.Name, nd.Oid, nd.Status, nd.Kind)

	mod_hash := fmt.Sprintf("%x", adler32.Checksum([]byte(mod_hash_string)))

	return mod_hash
}

func omfy_node(node *gosmi.SmiNode) OMFNode {
	omf_type := omfy_type(node.Type)

	omf_node := OMFNode{
		Access:      node.Access.String(),
		Decl:        node.Decl.String(),
		Kind:        node.Kind.String(),
		Name:        node.Name,
		Module:      node.GetModule().Module.Name,
		Description: node.Description,
		Oid:         node.Oid.String(),
		Status:      node.Status.String(),
		Type:        &omf_type,
	}

	omf_node.NodeHash = generate_node_hash(&omf_node)

	return omf_node
}

func omfy_imports(imports []models.Import) []OMFImport {
	var omf_imports []OMFImport

	import_map := join_imports(imports)

	for mod_name, imported_members := range import_map {

		var members []string

		for _, member_name := range imported_members {

			members = append(members, member_name)
		}

		new_mod_import := OMFImport{
			ModName:       mod_name,
			ImportedNodes: members,
		}

		omf_imports = append(omf_imports, new_mod_import)
	}

	return omf_imports
}

func omfy_table_columns(column_order []string, columns map[string]gosmi.SmiNode) []OMFNode {
	var omf_columns []OMFNode

	for _, col_name := range column_order {
		col_node := columns[col_name]

		new_col := omfy_node(&col_node)

		omf_columns = append(omf_columns, new_col)
	}

	return omf_columns
}

func omfy_table(tb *gosmi.Table) OMFTable {
	omf_table_node := omfy_node(&tb.SmiNode)

	omf_table_columns := omfy_table_columns(tb.ColumnOrder, tb.Columns)

	var omf_table_indexes []OMFIndex

	for _, index := range tb.Index {
		new_omf_index := omfy_node(&index)

		omf_table_indexes = append(omf_table_indexes, OMFIndex{OMFNode: new_omf_index})
	}

	return OMFTable{
		OMFNode: omf_table_node,
		Indexes: omf_table_indexes,
		Columns: omf_table_columns,
	}
}

func omfy_notification(nf *gosmi.Notification) OMFNotification {
	var omf_notification_objects []OMFNode

	for _, obj := range nf.Objects {
		new_obj := omfy_node(&obj)

		omf_notification_objects = append(omf_notification_objects, new_obj)
	}

	notification_obj := omfy_node(&nf.SmiNode)

	return OMFNotification{
		OMFNode: notification_obj,
		Objects: omf_notification_objects,
	}
}

func omfy_revisions(revs []models.Revision) []OMFRevision {
	var revs_arr []OMFRevision

	for _, rev := range revs {

		omf_rev := OMFRevision{
			Date:        rev.Date,
			Description: rev.Description,
		}

		revs_arr = append(revs_arr, omf_rev)
	}

	return revs_arr
}

func find_table_name_by_oid(oid string, tables *map[string]OMFTable) string {
	for idx, tb := range *tables {

		if tb.Oid == oid {
			return idx
		}

	}

	return ""
}

func collapse_map[T OMFTypeConstraint](mp map[string]T, order []string) []T {
	var res []T

	if len(order) > 0 {
		for _, key := range order {
			if entry, ok := mp[key]; ok {
				res = append(res, entry)
			}
		}

		return res
	}

	for _, entry := range mp {
		res = append(res, entry)
	}

	return res
}

func generate_module_hash(description string, name string, revisions *[]OMFRevision) string {
	rev_str := ""

	for _, rev := range *revisions {
		rev_str += rev.Description
		rev_str += rev.Date.String()
	}

	mod_hash_string := fmt.Sprintf("%s%s%s", description, name, rev_str)

	mod_hash := fmt.Sprintf("%x", md5.Sum([]byte(mod_hash_string)))

	return mod_hash
}

func extract_textual_convention_from_node(nd OMFNode) (OMFTextualConvention, bool) {
	if nd.Type.Decl == "TextualConvention" {
		return OMFTextualConvention{OMFType: *nd.Type}, true
	}

	return OMFTextualConvention{}, false
}

func extract_textual_convention_from_nodes(nds *[]OMFNode) []OMFTextualConvention {
	var tcs []OMFTextualConvention

	for _, nd := range *nds {
		tc, found := extract_textual_convention_from_node(nd)

		if found {
			tcs = append(tcs, tc)
		}
	}

	return tcs
}

func append_textual_convention(tc OMFTextualConvention, tc_map map[string]OMFTextualConvention) map[string]OMFTextualConvention {
	exits := tc_map[tc.Name]

	if exits.Name == tc.Name {
		return tc_map
	}

	tc_map[tc.Name] = tc

	return tc_map
}

func append_textual_conventions(tcs []OMFTextualConvention, tc_map map[string]OMFTextualConvention) map[string]OMFTextualConvention {
	for _, tc := range tcs {
		tc_map = append_textual_convention(tc, tc_map)
	}

	return tc_map
}

func get_tree_node_from_node(nd *gosmi.SmiNode, visited map[string]bool) (OMFTreeNode, map[string]bool) {
	omf_node := omfy_node(nd)

	new_tree_node := OMFTreeNode{
		NodeOid:  omf_node.Oid,
		Node:     omf_node,
		Children: make(map[string]*OMFTreeNode),
	}

	node_sub_tree := nd.GetSubtree()

	visited[nd.Name] = true

	if len(node_sub_tree) <= 0 {
		return new_tree_node, visited
	}

	for _, internal_node := range node_sub_tree {

		if visited[internal_node.Name] {
			continue
		}

		internal_node_sub_idx := internal_node.Oid[len(internal_node.Oid)-1]

		new_node_sub_tree, new_visited := get_tree_node_from_node(&internal_node, visited)

		new_tree_node.Children[fmt.Sprintf("%v", internal_node_sub_idx)] = &new_node_sub_tree

		visited = new_visited
	}

	return new_tree_node, visited
}

func find_root_of_nodes_list(nds []gosmi.SmiNode) OMFNode {
	smallest_oid_len := 123456789

	node_closest_to_root := gosmi.SmiNode{}

	for _, nd := range nds {
		if nd.Oid[0] == 0 {
			continue
		}

		if nd.OidLen < smallest_oid_len {
			smallest_oid_len = nd.OidLen

			node_closest_to_root = nd
		}
	}

	return omfy_node(&node_closest_to_root)
}

func get_tree_by_node_collection(nodes []gosmi.SmiNode) OMFTreeNode {
	root_of_nodes := find_root_of_nodes_list(nodes)

	visited := make(map[string]bool)

	root_tree_node := OMFTreeNode{
		NodeOid:  root_of_nodes.Oid,
		Node:     root_of_nodes,
		Children: make(map[string]*OMFTreeNode),
	}

	visited[root_of_nodes.Name] = true

	for _, node := range nodes {
		if visited[node.Name] {
			continue
		}

		node_sub_idx := node.Oid[len(node.Oid)-1]

		new_node, new_visited := get_tree_node_from_node(&node, visited)

		root_tree_node.Children[fmt.Sprintf("%v", node_sub_idx)] = &new_node

		visited = new_visited
	}

	return root_tree_node
}

func create_omf_tree(mod *gosmi.SmiModule) (OMFTree, error) {
	revisions := mod.GetRevisions()

	omfied_revisions := omfy_revisions(revisions)

	omf_imports := omfy_imports(mod.GetImports())

	mod_hash := generate_module_hash(mod.Description, mod.Name, &omfied_revisions)

	mod_tree := OMFTree{
		ModuleHash:    mod_hash,
		ContactInfo:   mod.ContactInfo,
		Description:   mod.Description,
		Language:      mod.Language.String(),
		Name:          mod.Name,
		Organization:  mod.Organization,
		Path:          mod.Path,
		Reference:     mod.Reference,
		ModuleImports: &omf_imports,
	}

	node_tree := get_tree_by_node_collection(mod.GetNodes())

	mod_tree.RootNode = &node_tree

	return mod_tree, nil
}

func omfy_module(mod *gosmi.SmiModule) OMFModule {
	nodes := mod.GetNodes()

	types := mod.GetTypes()

	imports := mod.GetImports()

	revisions := mod.GetRevisions()

	omfied_import := omfy_imports(imports)

	omfied_revisions := omfy_revisions(revisions)

	omf_tables_map := make(map[string]OMFTable)

	omf_notifications_map := make(map[string]OMFNotification)

	omf_scalars_map := make(map[string]OMFScalar)

	omf_types_map := make(map[string]OMFType)

	omf_other_nodes_map := make(map[string]OMFNode)

	module_hash := generate_module_hash(mod.Description, mod.Name, &omfied_revisions)

	var omf_tables_map_order []string

	var omf_notifications_map_order []string

	var omf_scalars_map_order []string

	var omf_types_map_order []string

	var omf_other_nodes_map_order []string

	omf_textual_conventions := make(map[string]OMFTextualConvention)

	for _, n := range nodes {
		if n.Kind == gosmi_types.NodeTable {
			tb := n.AsTable()

			omified_table := omfy_table(&tb)

			omf_tables_map[omified_table.Name] = omified_table

			omf_tables_map_order = append(omf_tables_map_order, omified_table.Name)

			tcs := extract_textual_convention_from_nodes(&omified_table.Columns)

			omf_textual_conventions = append_textual_conventions(tcs, omf_textual_conventions)

			continue
		}

		if n.Kind == gosmi_types.NodeNotification {
			nf := n.AsNotification()

			omfied_notification := omfy_notification(&nf)

			omf_notifications_map[omfied_notification.Name] = omfied_notification

			omf_notifications_map_order = append(omf_notifications_map_order, omfied_notification.Name)

			tcs := extract_textual_convention_from_nodes(&omfied_notification.Objects)

			omf_textual_conventions = append_textual_conventions(tcs, omf_textual_conventions)

			continue
		}

		if n.Kind == gosmi_types.NodeScalar {
			omfied_scalar := omfy_node(&n)

			omf_scalars_map[omfied_scalar.Name] = OMFScalar{OMFNode: omfied_scalar}

			omf_scalars_map_order = append(omf_scalars_map_order, omfied_scalar.Name)

			tc, found := extract_textual_convention_from_node(omfied_scalar)

			if found {
				omf_textual_conventions = append_textual_convention(tc, omf_textual_conventions)
			}

			continue
		}

		if n.Kind == gosmi_types.NodeRow {
			omfied_row := omfy_node(&n)

			table_oid := omfied_row.Oid[0 : len(omfied_row.Oid)-2]

			table_name := find_table_name_by_oid(table_oid, &omf_tables_map)

			if table, ok := omf_tables_map[table_name]; ok {

				table.Entry = omfied_row

				omf_tables_map[table_name] = table

			}

			continue
		}

		omf_other_node := omfy_node(&n)

		omf_other_nodes_map[omf_other_node.Name] = omf_other_node

		tc, found := extract_textual_convention_from_node(omf_other_node)

		if found {
			omf_textual_conventions = append_textual_convention(tc, omf_textual_conventions)
		}

		omf_other_nodes_map_order = append(omf_other_nodes_map_order, omf_other_node.Name)
	}

	for _, t := range types {
		omfied_type := omfy_type(&t.Type)

		omf_types_map[omfied_type.Name] = omfied_type

		omf_types_map_order = append(omf_types_map_order, omfied_type.Name)

		if omfied_type.Decl == "TextualConvention" {
			omf_textual_conventions = append_textual_convention(OMFTextualConvention{OMFType: omfied_type}, omf_textual_conventions)
		}
	}

	return OMFModule{
		ModuleHash:         module_hash,
		ContactInfo:        mod.ContactInfo,
		Description:        mod.Description,
		Language:           mod.Language.String(),
		Name:               mod.Name,
		Organization:       mod.Organization,
		Path:               mod.Path,
		Reference:          mod.Reference,
		Imports:            omfied_import,
		Revisions:          omfied_revisions,
		TextualConventions: collapse_map(omf_textual_conventions, []string{}),
		Scalars:            collapse_map(omf_scalars_map, omf_scalars_map_order),
		Indexes:            make([]OMFIndex, 1),
		Types:              collapse_map(omf_types_map, omf_types_map_order),
		Tables:             collapse_map(omf_tables_map, omf_tables_map_order),
		Notifications:      collapse_map(omf_notifications_map, omf_notifications_map_order),
		OtherNodes:         collapse_map(omf_other_nodes_map, omf_other_nodes_map_order),
	}
}

func module_trees(module_name string) {
	m, err := gosmi.GetModule(module_name)
	if err != nil {
		return
	}

	omf_module := omfy_module(&m)

	tl, err2 := toml.Marshal(omf_module)

	if err2 != nil {
		fmt.Printf("%s", err2)
		return
	}

	os.WriteFile("teste3.toml", tl, 0666)
}

func GetOmfCommomStruct(path string, module_name string, parseBack bool) (OMFModule, error) {

	init_gosmi(path, module_name)

	m, err := gosmi.GetModule(module_name)

	omf_module := omfy_module(&m)

	if err != nil {
		fmt.Printf("ModuleTrees Error: %s\n", err)
		return omf_module, err
	}

	exit_gosmi()
	return omf_module, nil
}

func GetOmfModuleTree(path string, module_name string) (OMFTree, error) {
	init_gosmi(path, module_name)

	m, err := gosmi.GetModule(module_name)

	if err != nil {
		fmt.Printf("GetOmfModuleTree Error: %v\n", err)

		return OMFTree{}, err
	}

	omf_tree, err2 := create_omf_tree(&m)

	if err2 != nil {
		fmt.Printf("create_omf_tree Error: %v\n", err2)

		return OMFTree{}, err
	}

	exit_gosmi()

	return omf_tree, nil
}
