package assembly

import (
	"encoding/json"
	"os"
	"sync"

	"t-api/entity"
	"t-api/internal/aes"
	"t-api/service"

	"github.com/pkg/errors"
)

type User struct {
	data    *entity.User
	service *service.Service
}

type Cache struct {
	m        *sync.Mutex
	key      []byte
	data     map[int64]User
	filepath string
}

func NewCache(filepath string, key []byte) (*Cache, error) {
	data, err := readFromFile(filepath, key)
	if err != nil {
		return nil, errors.WithMessage(err, "read users data from file")
	}

	return &Cache{
		m:        &sync.Mutex{},
		key:      key,
		data:     data,
		filepath: filepath,
	}, nil
}

func (c *Cache) Get(id int64) (User, bool) {
	c.m.Lock()
	defer c.m.Unlock()

	user, found := c.data[id]
	return user, found
}

func (c *Cache) GetAll() []User {
	c.m.Lock()
	defer c.m.Unlock()

	all := make([]User, 0)
	for _, user := range c.data {
		all = append(all, user)
	}

	return all
}

func (c *Cache) Add(id int64, user User) {
	c.m.Lock()
	defer c.m.Unlock()

	c.data[id] = user
}

func (c *Cache) Update(id int64, newUser User) {
	c.m.Lock()
	defer c.m.Unlock()

	_, found := c.data[id]
	if !found {
		return
	}

	c.data[id] = newUser
}

func readFromFile(path string, key []byte) (map[int64]User, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, errors.WithMessage(err, "open file")
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, errors.WithMessage(err, "stat file")
	}

	fileBytes := make([]byte, stat.Size())
	_, err = file.Read(fileBytes)
	if err != nil {
		return nil, errors.WithMessage(err, "read file bytes")
	}

	users := make(map[int64]User)
	if len(fileBytes) == 0 {
		return users, nil
	}

	decrypted, err := aes.Decrypt(key, fileBytes)
	if err != nil {
		return nil, errors.WithMessage(err, "encrypt json")
	}

	usersData := make([]*entity.User, 0)
	err = json.Unmarshal(decrypted, &usersData)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshal json")
	}

	for _, data := range usersData {
		users[data.ChatId] = User{
			data:    data,
			service: nil,
		}
	}

	return users, nil
}

func (c *Cache) SaveToFile() error {
	file, err := os.OpenFile(c.filepath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return errors.WithMessage(err, "open file")
	}
	defer file.Close()

	usersData := make([]*entity.User, 0)
	for _, data := range c.data {
		usersData = append(usersData, data.data)
	}

	dataBytes, err := json.Marshal(usersData)
	if err != nil {
		return errors.WithMessage(err, "marshal json")
	}

	encrypted, err := aes.Encrypt(c.key, dataBytes)
	if err != nil {
		return errors.WithMessage(err, "encrypt json")
	}

	_, err = file.Write(encrypted)
	if err != nil {
		return errors.WithMessage(err, "write file")
	}

	return nil
}
