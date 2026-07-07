package parser

type File struct {
	Name string `json:"name"`
	Url string 	`json:"url"`
}

type Keyword struct {
	Id int16 	`json:"id"`
	Name string `json:"name"`
}


type Result struct {
	Id string			 `json:"id"`
	Name string 		 `json:"name"`
	DatePublished string `json:"date_published"`
	Files []File 		 `json:"files,omitempty"`
	Keywords []Keyword	 `json:"keywords,omitempty"`
}

type Data struct {
	Results []Result
}

type ChannelPayload struct {
	Url string
	FilePath string
}

type SuccessPayload struct {
	Url string
	Page int
}
