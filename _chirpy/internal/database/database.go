package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

const DbPath = "database.json"

type Db struct {
	path   string
	Chirps map[string]Chirp `json:"chirps"`
	Users  map[string]User  `json:"users"`
	mu     sync.Mutex
}

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

type User struct {
	Id       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewDb(path string) Db {
	return Db{path: path, Chirps: make(map[string]Chirp), Users: make(map[string]User)}
}

func (d *Db) Init() error {
	_, err := os.Open(d.path)
	if errors.Is(err, fs.ErrNotExist) {
		dat, err := json.Marshal(d)
		if err != nil {
			return errors.New("error encoding chirps")
		}
		if err = os.WriteFile(d.path, dat, 0777); err != nil {
			return errors.New("error creating db file")
		}
	}
	return nil
}

func (d *Db) Read() (struct {
	Chirps []Chirp
	Users  []User
}, error) {
	if err := d.Init(); err != nil {
		return struct {
			Chirps []Chirp
			Users  []User
		}{}, errors.New(err.Error())
	}
	data, err := os.ReadFile(d.path)
	if err != nil {
		return struct {
			Chirps []Chirp
			Users  []User
		}{}, errors.New("error opening file")
	}
	if err = json.Unmarshal(data, d); err != nil {
		fmt.Print(err.Error())
		return struct {
			Chirps []Chirp
			Users  []User
		}{}, errors.New("error unmarshaling data")
	}
	output := struct {
		Chirps []Chirp
		Users  []User
	}{Chirps: make([]Chirp, 10), Users: make([]User, 10)}
	d.mu.Lock()
	for _, value := range d.Chirps {
		output.Chirps = append(output.Chirps, value)
	}
	for _, value := range d.Users {
		output.Users = append(output.Users, value)
	}
	d.mu.Unlock()
	return output, nil
}

func (d *Db) SaveUser(data struct {
	Email    string
	Password string
}) (User, error) {
	if _, err := d.Read(); err != nil {
		return User{}, errors.New("error reading from db")
	}
	usr, _ := d.GetUserByEmail(data.Email)
	if usr.Email == data.Email {
		return User{}, errors.New("user already exists")
	}
	user := User{Id: d.GetLatestIdUsers() + 1, Email: data.Email, Password: data.Password}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Users[fmt.Sprint(user.Id)] = user
	// write to db
	file, err := os.Create(d.path)
	if err != nil {
		return User{}, errors.New("error creating db file")
	}
	defer file.Close()
	dat, err := json.Marshal(d)
	if err != nil {
		return User{}, errors.New("error encoding chirps")
	}
	if _, err = file.Write(dat); err != nil {
		return User{}, errors.New("error saving chirp")
	}
	return user, nil
}

func (d *Db) SaveChirp(data struct{ Body string }) (Chirp, error) {
	if _, err := d.Read(); err != nil {
		return Chirp{}, errors.New("error reading from db")
	}
	chirp := Chirp{Id: d.GetLatestIdChirp() + 1, Body: data.Body}
	d.mu.Lock()
	d.Chirps[fmt.Sprint(chirp.Id)] = chirp
	// write to db
	file, err := os.Create(d.path)
	if err != nil {
		return Chirp{}, errors.New("error creating db file")
	}
	defer file.Close()
	dat, err := json.Marshal(d)
	d.mu.Unlock()
	if err != nil {
		return Chirp{}, errors.New("error encoding chirps")
	}
	if _, err = file.Write(dat); err != nil {
		return Chirp{}, errors.New("error saving chirp")
	}
	return chirp, nil
}

func (d *Db) GetChirp(id int) (Chirp, error) {
	data, err := d.Read()
	if err != nil {
		return Chirp{}, errors.New("error reading db file")
	}
	for _, value := range data.Chirps {
		if id == value.Id {
			return value, nil
		}
	}
	return Chirp{}, nil
}

func (d *Db) GetLatestIdChirp() int {
	max := 0
	d.mu.Lock()
	for _, value := range d.Chirps {
		if value.Id >= max {
			max = value.Id
		}
	}
	d.mu.Unlock()
	return max
}

func (d *Db) GetLatestIdUsers() int {
	max := 0
	d.mu.Lock()
	for _, value := range d.Users {
		if value.Id >= max {
			max = value.Id
		}
	}
	d.mu.Unlock()
	return max
}

func (d *Db) GetUserByEmail(email string) (User, error) {
	data, err := d.Read()
	if err != nil {
		return User{}, errors.New("error reading db file")
	}
	var usr User
	for _, value := range data.Users {
		if value.Email == email {
			usr = value
		}
	}
	return usr, nil
}

func (d *Db) UpdateUser(id int, data struct {
	Email    string
	Password string
}) (User, error) {
	usr, err := d.GetUserById(id)
	if err != nil {
		return User{}, errors.New("error reading db")
	}
	if usr == nil {
		return User{}, nil
	}
	usr.Email = data.Email
	hashedPass, _ := bcrypt.GenerateFromPassword([]byte(data.Password), 2)
	usr.Password = string(hashedPass)
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Users[fmt.Sprint(id)] = *usr
	dat, _ := json.Marshal(d)
	file, _ := os.Create(d.path)
	if _, err = file.Write(dat); err != nil {
		return User{}, errors.New("error saving user")
	}
	return *usr, nil
}

func (d *Db) GetUserById(id int) (*User, error) {
	data, err := d.Read()
	if err != nil {
		return &User{}, errors.New("error reaeding db file")
	}
	var usr User
	for _, value := range data.Users {
		if value.Id == id {
			usr = value
		}
	}
	return &usr, nil
}
