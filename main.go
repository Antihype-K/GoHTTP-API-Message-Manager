package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var maper map[int]string = make(map[int]string)
var mtx sync.RWMutex

func sliceStringToInt(sliceString []string) []int {
	sliceInt := make([]int, 0)
	for _, split := range sliceString {
		split, err := strconv.Atoi(split)
		if err != nil {
			fmt.Println("Ошибка конвертирования ", err.Error())
			return sliceInt
		} else {
			sliceInt = append(sliceInt, split)
		}
	}
	return sliceInt
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	httpRequestBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		fmt.Println("Ошибка при прочтении запроса ", err)
		return
	}
	if len(httpRequestBody) == 0 {
		w.Write([]byte("Отправьте ключи для удаления (числа через пробел)\n"))
		return
	}
	stringes := string(httpRequestBody)
	splits := strings.Fields(stringes)
	splitInt := sliceStringToInt(splits)

	mtx.Lock()
	var mapDelete map[int]string = make(map[int]string)
	var notFound = make([]int, 0)
	for _, split := range splitInt {
		if value, exists := maper[split]; exists {
			delete(maper, split)
			mapDelete[split] = value
		} else {
			notFound = append(notFound, split)
		}
	}

	if len(mapDelete) > 0 {
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Println("Удаленные элементы")
		for key, value := range mapDelete {
			fmt.Println("Удален ключ ", key, "со значением ", value)
			fmt.Println("\n" + strings.Repeat("=", 50))
		}
		if len(notFound) > 0 {
			fmt.Println("Ключ ", notFound, "не найден")
		}
		fmt.Println("В мапе осталось ", len(maper), " ключей")
	}
	mtx.Unlock()

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"success":   true,
		"deleted":   mapDelete,
		"not_found": notFound,
		"total":     len(maper),
		"message":   fmt.Sprintf("Удалено: %d, Не найдено: %d", len(mapDelete), len(notFound)),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Println("Ошибка отправки JSON:", err)
	}

}

func addHandler(w http.ResponseWriter, r *http.Request) {
	httpRequestBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, "Ошибка при прочтении http тела", http.StatusBadRequest)
		fmt.Println("Ошибка при прочтении http тела", err)
		return
	}

	stringes := string(httpRequestBody)
	splits := strings.Fields(stringes)
	mtx.Lock()

	var mapAdd map[int]string = make(map[int]string)
	maxID := 0
	for id := range maper {
		if id > maxID {
			maxID = id
		}
	}

	for i, split := range splits {
		key := maxID + i + 1
		maper[key] = split
		mapAdd[key] = split
	}
	mtx.Unlock()
	if len(mapAdd) > 0 {
		for key, value := range mapAdd {
			fmt.Println("Добавлено значение ", value, "с ключом ", key)
		}
	}
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"success": true,
		"added":   mapAdd,
		"total":   len(maper),
		"message": fmt.Sprintf("Добавлено %d элементов", len(mapAdd)),
	}

	json.NewEncoder(w).Encode(response)
}

func printHandler(w http.ResponseWriter, r *http.Request) {
	mtx.RLock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(maper)
	fmt.Println(maper)
	mtx.RUnlock()
}

func getByID(w http.ResponseWriter, r *http.Request) {
	httpRequestBody, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, "Ошибка при прочтении http тела", http.StatusBadRequest)
		fmt.Println("Ошибка при прочтении http тела", err)
		return
	}

	if len(httpRequestBody) == 0 {
		w.Write([]byte("Отправьте ключи для получения значений (числа через пробел)\n"))
		return
	}
	stringes := string(httpRequestBody)
	splits := strings.Fields(stringes)
	splitInt := sliceStringToInt(splits)

	infoID := make(map[int]string)
	notFound := make([]int, 0)
	mtx.RLock()
	defer mtx.RUnlock()
	for _, id := range splitInt {
		if value, exists := maper[id]; exists {
			infoID[id] = value
			fmt.Printf("Найдено: ID=%d -> %s\n", id, value)
		} else {
			w.WriteHeader(http.StatusNotFound)
			notFound = append(notFound, id)

		}

	}
	w.Header().Set("Content-Type", "application/json")

	if len(notFound) > 0 {
		// Если есть ненайденные - отправляем 404
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":     "Некоторые сообщения не найдены",
			"found":     infoID,
			"not_found": notFound,
			"total":     len(maper),
		})
	} else {
		// Если все найдены - отправляем 200
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"found":   infoID,
			"total":   len(infoID),
		})
	}

}

func main() {
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/print", printHandler)
	http.HandleFunc("/get", getByID)
	err := http.ListenAndServe(":9069", nil)
	if err != nil {
		fmt.Println("404", err.Error())
	}
}
