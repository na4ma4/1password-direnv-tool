package model

type SecretRef struct {
	Value string    `json:"value"`
	Files *FileList `json:"files,omitempty"`
}

func (s *SecretRef) GetValue() string {
	return s.Value
}

func (s *SecretRef) GetFiles() *FileList {
	return s.Files
}

func (s *SecretRef) MergeFiles(files *FileList) {
	if s.Files == nil {
		s.Files = files
	} else {
		s.Files.Merge(files)
	}
}
