package main

type Text struct {
	Index   int    `json:"index:"`
	Content string `json:"content"`
}

type Person struct {
	Name      string `json:"name"`
	Age       int    `json:"age"`
	Gender    string `json:"gender"`
	PatientID int    `json:"int"`
}
