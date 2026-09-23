package main

import (
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("не найдено")

type ValidationError struct {
	Field string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("поле %q не прошло валидацию", e.Field)
}

// queryDB — нижний слой: возвращает ErrNotFound, обёрнутый своим контекстом.
func queryDB(id int) (string, error) {
	if id != 42 {
		return "", fmt.Errorf("запрос строки id=%d: %w", id, ErrNotFound)
	}
	return "Alice", nil
}

func lookup(id int) (string, error) {
	if id < 0 {
		return "", &ValidationError{Field: "id"}
	}
	name, err := queryDB(id)
	if err != nil {
		// %w вкладывает ошибку queryDB внутрь — контекст каждого слоя сохраняется
		return "", fmt.Errorf("lookup id=%d: %w", id, err)
	}
	return name, nil
}

// mustLookup паникует вместо возврата ошибки. Так оформляют только то,
// что при нормальной работе программы произойти не должно: панику здесь
// провоцирует баг вызывающего кода, а не ожидаемое «нет такой строки».
func mustLookup(id int) string {
	name, err := lookup(id)
	if err != nil {
		panic(err)
	}
	return name
}

// safeLookup ставит барьер на границе: recover работает только внутри
// defer и только там ловит панику. Пойманное значение превращаем обратно
// в error через именованный результат err, чтобы наружу паника не ушла.
func safeLookup(id int) (name string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("паника в mustLookup(id=%d): %v", id, r)
		}
	}()
	return mustLookup(id), nil
}

func main() {
	for _, id := range []int{42, 7, -1} {
		name, err := lookup(id)
		switch {
		case err == nil:
			fmt.Println("ок:", name)
		case errors.Is(err, ErrNotFound):
			fmt.Println("не найдено:", err)
		default:
			var ve *ValidationError
			if errors.As(err, &ve) {
				fmt.Println("ошибка валидации, поле:", ve.Field)
			} else {
				fmt.Println("другая ошибка:", err)
			}
		}
	}

	// panic + recover: mustLookup(7) паникует, safeLookup гасит панику и
	// отдаёт обычную ошибку — дальше её обрабатывают как всегда.
	if _, err := safeLookup(7); err != nil {
		fmt.Println("восстановились после паники:", err)
	}
}
