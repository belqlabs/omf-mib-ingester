package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"omf-mibParser/omifier"
	"os"
)

type arrayStrings []string

func main() {
	omf_tree, err := omifier.GetOmfModuleTree("/home/augusto-sigolo/Belqlabs/mibs", "ZTE-AN-GPON-REMOTE-ONU-MIB")

	if err != nil {
		fmt.Printf("Erro na main\n")
		fmt.Printf("%v\n", err)
	}

	omf_complete_tree, err := omifier.CreateCompleteTreeFromModule("/home/augusto-sigolo/Belqlabs/mibs", "ZTE-AN-GPON-REMOTE-ONU-MIB")

	if err != nil {
		fmt.Printf("Erro na main\n")
		fmt.Printf("%v\n", err)
	}

	tree_json, err2 := json.Marshal(omf_tree)

	if err2 != nil {
		fmt.Printf("Erro no marshal\n")
		fmt.Printf("%v\n", err2)
		return
	}

	complete_tree_json, err2 := json.Marshal(omf_complete_tree)

	if err2 != nil {
		fmt.Printf("Erro no marshal\n")
		fmt.Printf("%v\n", err2)
		return
	}

	var out bytes.Buffer
	json.Indent(&out, tree_json, "", "  ")

	os.WriteFile("teste-tree-zte-smi.json", out.Bytes(), 0666)

	out.Reset()
	json.Indent(&out, complete_tree_json, "", " ")

	os.WriteFile("teste-complete-tree-zte-smi.json", out.Bytes(), 0666)
}
