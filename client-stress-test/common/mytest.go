package common

import (
	"encoding/json"
	"fmt"
	"github.com/mattbaird/jsonpatch"
	"io/ioutil"
)

// User representa a estrutura de um usuário com ID e Nome.
type UserX struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func printPatch(oldList []interface{}, newList []interface{}) []jsonpatch.JsonPatchOperation {
	oldListJson, _ := json.Marshal(oldList)
	newListJson, _ := json.Marshal(newList)

	patches, _ := jsonpatch.CreatePatch(oldListJson, newListJson)
	patchBytes, _ := json.MarshalIndent(patches, "", "  ")

	fmt.Printf("Result: %s\n", string(patchBytes))

	return patches
}

func RunPatchTest() {

	var oldList []map[string]interface{}
	fileContentBefore, lastContentErr := ioutil.ReadFile("test/users-test2-1.json")
	if lastContentErr == nil && string(fileContentBefore) != "" {
		fmt.Println(string(fileContentBefore))

		if err := json.Unmarshal([]byte(fileContentBefore), &oldList); err != nil {
			panic(err)
		}
		//fmt.Println(oldList)
	}

	var newList []map[string]interface{}
	fileContentAfter, lastContentErr := ioutil.ReadFile("test/users-test2-2.json")
	if lastContentErr == nil && string(fileContentAfter) != "" {
		fmt.Println(string(fileContentAfter))

		if err := json.Unmarshal([]byte(fileContentAfter), &newList); err != nil {
			panic(err)
		}
		//fmt.Println(newList)
	}

	//newArray := []map[string]interface{}{}
	//for a := range newList {
	//	newData := map[string]interface{}{}
	//	newData["userId"] = newList[a]["userId"]
	//	newData["name"] = newList[a]["name"]
	//	newArray = append(newArray, newData)
	//}
	//
	//a, _ := json.Marshal(newArray)
	//fmt.Println(string(a))
	//if 1 == 1 {
	//	return
	//}

	//
	//oldList := []map[string]interface{}{
	//	{"userId": "0", "Name": "Joe"},
	//	{"userId": "1", "Name": "Alice"},
	//	{"userId": "2", "Name": "Bob"},
	//	{"userId": "3", "Name": "Charlie"},
	//	{"userId": "4", "Name": "Daniel"},
	//}
	//
	//newList := []map[string]interface{}{
	//	{"userId": "1", "Name": "Alice"},
	//	{"userId": "2", "Name": "Bob"},
	//	{"userId": "3", "Name": "Charlie"},
	//	{"userId": "4", "Name": "Daniel"},
	//	{"userId": "0", "Name": "Joe"},
	//	{"userId": "5", "Name": "Joe"},
	//}

	if ValidateIfShouldUseCustomJsonPatch(fileContentBefore, fileContentAfter, "userId") {
		fmt.Println("Yes!!")
		myPatch := CreateJsonPatchFromMaps(oldList, newList, "userId")
		//myPatchJson, _ := json.MarshalIndent(myPatch, "", "  ")
		fmt.Println("Patch: ", string(myPatch))
	} else {
		fmt.Println("No!!")
	}

	//patch, err := jsondiff.Compare(oldList, newList)
	//if err != nil {
	//	// handle error
	//}
	//b, err := json.MarshalIndent(patch, "", "    ")
	//if err != nil {
	//	// handle error
	//}
	//os.Stdout.Write(b)
	//
	//if 1 == 1 {
	//	return
	//}

	// Print first result
	//printPatch(oldList, newList)

	// Add an empty UserX to newList and print second result
	//newList = append(newList, UserX{})
	//printPatch(oldList, newList)

	//oldListAsGeneric, _ := oldList.([]map[string]interface{})
	//
	//printPatchGt(oldList, newList)

	//
	//patchA, err := GeneratePatch(oldUserXs, newerUserXs)
	//if err != nil {
	//	fmt.Printf("Erro ao gerar patch: %v\n", err)
	//	return
	//}
	//
	//// Imprime o patch gerado.
	//fmt.Printf("Patch (como seria antes):\n")
	//fmt.Println(patchA)
	//
	//newerUserXs = append(newerUserXs, UserX{})
	//
	//patchB, err := GeneratePatch(oldUserXs, newerUserXs)
	//if err != nil {
	//	fmt.Printf("Erro ao gerar patch: %v\n", err)
	//	return
	//}
	//
	//// Imprime o patch gerado.
	//fmt.Println("")
	//fmt.Printf("Patch (new):\n")
	//fmt.Println(patchB)

	//patch, err := GeneratePatch(oldUserXs, newUserXs)
	//if err != nil {
	//	fmt.Printf("Erro ao gerar patch: %v\n", err)
	//	return
	//}

	// Imprime o patch gerado.
	//fmt.Printf("Patch fase 1: %s\n", patch)

	//patchB, err := GeneratePatch(newUserXs, newerUserXs)
	//if err != nil {
	//	fmt.Printf("Erro ao gerar patch: %v\n", err)
	//	return
	//}

	// Imprime o patch gerado.
	//fmt.Printf("Patch fase 2: %s\n", patchB)

	//obj1, _ := json.Marshal(oldUserXs)
	//obj2, _ := json.Marshal(newerUserXs)
	//
	//fmt.Println(string(obj1))
	//fmt.Println(string(obj2))
	//
	//evanphxPatch, err := teste.CreateMergePatch(obj1, obj2)
	//if err != nil {
	//	fmt.Println("Error creating merge patch:", err)
	//	return
	//}

	// Print the generated JSON Patch
	//fmt.Println("Generated JSON Patch by evanphxPatch:", string(evanphxPatch))
	//
	//fmt.Println("")
	//
	//snorwinPatch, _ := testeb.CreateJSONPatch(newerUserXs, oldUserXs)
	////fmt.Println(snorwinPatch.String())
	//
	//fmt.Println("Generated JSON Patch by evanphxPatch:")
	//fmt.Println(snorwinPatch.String())

}

// GeneratePatch gera um patch JSON representando as diferenças entre duas listas de usuários.
//func GeneratePatch(oldUserXs, newUserXs []UserX) (string, error) {
//	oldBytes, err := json.Marshal(oldUserXs)
//	if err != nil {
//		return "", err
//	}
//
//	newBytes, err := json.Marshal(newUserXs)
//	if err != nil {
//		return "", err
//	}
//
//	patches, err := jsonpatch.CreatePatch(oldBytes, newBytes)
//	if err != nil {
//		return "", err
//	}
//
//	patchBytes, err := json.Marshal(patches)
//	if err != nil {
//		return "", err
//	}
//
//	return string(patchBytes), nil
//}
