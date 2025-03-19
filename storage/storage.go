package storage

import (
	"fmt"
	"log/slog"
	"path"
	"sort"
	"sync"
)

const (
	usersFile = "users.json"
	filmsFile = "films.json"
	utilFile  = "util.json"
)

type Storage struct {
	users map[int64]UserInfo
	films map[int]FilmInfo
	util  util

	usersPath string
	filmsPath string
	utilPath  string

	usersMu sync.RWMutex
	filmsMu sync.RWMutex
	utilMu  sync.RWMutex
}

type UserInfo struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Vote     int    `json:"vote"`
}

type FilmInfo struct {
	Name  string `json:"name"`
	Added int64  `json:"added_by"`
}

type FilmStat struct {
	Id      int
	Name    string
	Votes   int
	Voters  []UserInfo
	AddedBy UserInfo
}

func New(dataPath string) (*Storage, error) {
	usersPath := path.Join(dataPath, usersFile)
	users := map[int64]UserInfo{}
	if err := loadFromFileJSON(usersPath, &users); err != nil {
		return nil, fmt.Errorf("failed to load users data: %w", err)
	}

	filmsPath := path.Join(dataPath, filmsFile)
	films := map[int]FilmInfo{}
	if err := loadFromFileJSON(filmsPath, &films); err != nil {
		return nil, fmt.Errorf("failed to load films data: %w", err)
	}

	utilPath := path.Join(dataPath, utilFile)
	u := util{}
	if err := loadFromFileJSON(utilPath, &u); err != nil {
		return nil, fmt.Errorf("faield to load util data: %w", err)
	}

	return &Storage{
		users:     users,
		films:     films,
		util:      u,
		usersPath: usersPath,
		filmsPath: filmsPath,
		utilPath:  utilPath,
	}, nil
}

func (s *Storage) Register(userID int64, name string, username string) (bool, error) {
	s.usersMu.RLock()
	_, ok := s.users[userID]
	s.usersMu.RUnlock()
	if ok {
		return false, nil
	}

	s.usersMu.Lock()
	defer s.usersMu.Unlock()
	s.users[userID] = UserInfo{
		Name:     name,
		Username: username,
	}
	if err := s.flushUsers(); err != nil {
		return false, fmt.Errorf("failed to write users data: %r", err)
	}

	return true, nil
}

func (s *Storage) Status() []FilmStat {
	if len(s.films) == 0 {
		return nil
	}

	stats := make([]FilmStat, 0, len(s.films))
	idx := map[int]int{}

	s.filmsMu.RLock()
	for filmID, info := range s.films {
		idx[filmID] = len(stats)
		stats = append(stats, FilmStat{Id: filmID, Name: info.Name})
	}
	s.filmsMu.RUnlock()

	s.usersMu.RLock()
	for _, info := range s.users {
		id, ok := idx[info.Vote]
		if ok {
			stats[id].Votes++
		}
	}
	s.usersMu.RUnlock()

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Votes > stats[j].Votes
	})

	return stats
}

func (s *Storage) StatusFull() []FilmStat {
	if len(s.films) == 0 {
		return nil
	}

	stats := make([]FilmStat, 0, len(s.films))
	idx := map[int]int{}

	s.filmsMu.RLock()
	for filmID, info := range s.films {
		idx[filmID] = len(stats)
		stats = append(stats, FilmStat{
			Id:      filmID,
			Name:    info.Name,
			AddedBy: s.GetUser(info.Added),
		})
	}
	s.filmsMu.RUnlock()

	s.usersMu.RLock()
	for _, info := range s.users {
		id, ok := idx[info.Vote]
		if ok {
			stats[id].Votes++
			stats[id].Voters = append(stats[id].Voters, info)
		}
	}
	s.usersMu.RUnlock()

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Votes > stats[j].Votes
	})

	return stats
}

func (s *Storage) AddFilm(userID int64, name string) error {
	s.filmsMu.Lock()
	defer s.filmsMu.Unlock()

	s.films[s.newID()] = FilmInfo{
		Name:  name,
		Added: userID,
	}

	return s.flushFilms()
}

func (s *Storage) RemoveFilm(name string) (bool, error) {
	s.filmsMu.Lock()
	defer s.filmsMu.Unlock()

	removed := false
	for id, info := range s.films {
		if info.Name == name {
			delete(s.films, id)
			if err := s.flushFilms(); err != nil {
				return false, err
			}
			removed = true
		}
	}

	return removed, nil
}

func (s *Storage) ResetVotes() {
	s.usersMu.Lock()
	defer s.usersMu.Unlock()

	for id, info := range s.users {
		info.Vote = 0
		s.users[id] = info
	}
}

func (s *Storage) Vote(userID int64, filmID int) (bool, error) {
	if filmID == 0 {
		s.usersMu.Lock()
		defer s.usersMu.Unlock()
		usr, ok := s.users[userID]
		if !ok {
			return false, fmt.Errorf("no userID=%d", userID)
		}
		usr.Vote = filmID
		s.users[userID] = usr
		return true, nil
	}

	s.filmsMu.RLock()
	_, ok := s.films[filmID]
	s.filmsMu.RUnlock()
	if !ok {
		return false, fmt.Errorf("no filmID=%d", filmID)
	}

	s.usersMu.Lock()
	defer s.usersMu.Unlock()
	usr, ok := s.users[userID]
	if !ok {
		return false, fmt.Errorf("no userID=%d", userID)
	}
	usr.Vote = filmID
	s.users[userID] = usr

	if err := s.flushUsers(); err != nil {
		return false, err
	}

	return true, nil
}

func (s *Storage) GetUser(userID int64) UserInfo {
	s.usersMu.RLock()
	defer s.usersMu.RUnlock()

	return s.users[userID]
}

func (s *Storage) GetVote(userID int64) int {
	s.usersMu.RLock()
	defer s.usersMu.RUnlock()

	return s.users[userID].Vote
}

func (s *Storage) SetMonitor(chatId int64, msgID int64) {
	slog.Debug(fmt.Sprintf("set monitor chat=%d msg=%d", chatId, msgID))
	s.utilMu.Lock()
	s.util.Monitor = Monitor{ChatId: chatId, MsgId: msgID}
	if err := s.flushUtil(); err != nil {
		slog.Error("failed to save util")
	}
	s.utilMu.Unlock()
}

func (s *Storage) GetMonitor() Monitor {
	s.utilMu.RLock()
	defer s.utilMu.RUnlock()
	return s.util.Monitor
}

// func (s *Storage) ClearMonitor() {
// 	s.utilMu.Lock()
// 	defer s.usersMu.Unlock()
// 	s.util.Monitor = Monitor{}
// }
