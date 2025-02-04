package main

import (
	"fmt"
	gosmi "github.com/belqlabs/omf-gosmi"
	gosmi_types "github.com/belqlabs/omf-gosmi/types"
)

type OMFCompleteModuleTree struct {
  Hash string
  Contact string
  Description string
  Language string
  Name string
  Organization string
  Path string
  Reference string
  Tree map[string]*OMFTreeNode
}

func append_module(module_name string) error {
	_, err := gosmi.LoadModule(module_name)

  return err
}

func get_leaf_by_oid(oid gosmi_types.Oid, node_map map[string]*OMFTreeNode) *OMFTreeNode {
  if len(oid) == 1 {
    _, leaf_exists := node_map[fmt.Sprintf("%v", oid)];

    if !leaf_exists {
      node_map[fmt.Sprintf("%v", oid)] = &OMFTreeNode{
        Children: make(map[string]*OMFTreeNode),
      }
    }

    return node_map[fmt.Sprintf("%v", oid)]
  }

  oid_root := fmt.Sprintf("%v", oid[0]);

  _, oid_root_exists := node_map[oid_root];

  if !oid_root_exists {
    node_map[oid_root] = &OMFTreeNode{
      Children: make(map[string]*OMFTreeNode),
    }
  }

  oid_rest := oid[1:];

  return get_leaf_by_oid(oid_rest, node_map[oid_root].Children)
}

func create_tree_from_node_list(nd_list []gosmi.SmiNode) map[string]*OMFTreeNode {
  tree_map := make(map[string]*OMFTreeNode);

  for _, node := range nd_list {

    clean_node := get_leaf_by_oid(node.Oid, tree_map)    

    clean_node.Node = omfy_node(&node)

    clean_node.NodeOid = node.Oid.String()

  }

  return tree_map;
}

func create_complete_tree_from_module (mod *gosmi.SmiModule) (OMFCompleteModuleTree, error) {
  provided_module_tree, tree_err := create_omf_tree(mod)
  
  if tree_err != nil {
    return OMFCompleteModuleTree{}, tree_err
  }

  omf_complete_module_tree := OMFCompleteModuleTree {
    Hash: provided_module_tree.ModuleHash,
    Contact: provided_module_tree.ContactInfo,
    Description: provided_module_tree.Description,
    Language: provided_module_tree.Language,
    Name: provided_module_tree.Name,
    Organization: provided_module_tree.Organization,
    Path: provided_module_tree.Path,
    Reference: provided_module_tree.Reference,
  }

  imported_module_trees := make(map[string]*OMFCompleteModuleTree)

  for _, imported_module := range *provided_module_tree.ModuleImports {
    
    _, visited := imported_module_trees[imported_module.ModName];

    if visited {
      continue
    }

    imported_module_err := append_module(imported_module.ModName)

    if imported_module_err != nil {
     return  omf_complete_module_tree, nil
    }
  }

  complete_node_collectoin := make([]gosmi.SmiNode, 0);

  for _, module := range gosmi.GetLoadedModules() {
    complete_node_collectoin = append(complete_node_collectoin, module.GetNodes()...)
  }

  complete_tree := create_tree_from_node_list(complete_node_collectoin)

  omf_complete_module_tree.Tree = complete_tree

  return omf_complete_module_tree, nil
}

func CreateCompleteTreeFromModule(path string, module_name string) (OMFCompleteModuleTree, error) {
	init_gosmi(path, module_name)

	m, err := gosmi.GetModule(module_name)

  if err != nil {
    return OMFCompleteModuleTree{}, err
  }

  return create_complete_tree_from_module(&m)
} 
