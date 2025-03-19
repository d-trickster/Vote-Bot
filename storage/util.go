package storage

import (
	"encoding/json"
	"log/slog"
	"os"
)

type util struct {
	IdCnt   int     `json:"id_cnt"`
	Monitor Monitor `json:"monitor"`
}

type Monitor struct {
	ChatId int64 `json:"chat_id"`
	MsgId  int64 `json:"message_id"`
}

func loadFromFileJSON(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewDecoder(f).Decode(v)
}

func (s *Storage) flushUsers() error {
	return saveToFileJSON(s.usersPath, s.users)
}

func (s *Storage) flushFilms() error {
	return saveToFileJSON(s.filmsPath, s.films)
}

func (s *Storage) flushUtil() error {
	return saveToFileJSON(s.utilPath, s.util)
}

func saveToFileJSON(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

func (s *Storage) newID() int {
	s.utilMu.Lock()
	s.util.IdCnt++
	id := s.util.IdCnt
	if err := s.flushUtil(); err != nil {
		slog.Error("failed to save util")
	}
	s.utilMu.Unlock()
	return id
}
