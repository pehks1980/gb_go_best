package filter_test

import (
	"fmt"
	"github.com/pehks1980/gb_go_best/kurs/app1/filter"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

// minimock -i DB укзывает что надо мокить (

//go install github.com/gojuno/minimock/v3/cmd/minimock # устанавали
//minimock -i DB # запускаем генерацию заглушки для интерфейса DB


func TestFilter_New(t *testing.T)  {
	// инициализция мокинг структур
	ParseMock := &filter.ParseMock{}

	//инициализация калькулятора с подстановкой мокинга в db
	//calc := &Calculator{db: dbMock}
	// настройка поведения функций черный ящик вход <-> результ
	cond := &filter.Condition{Colname:"a", Oper: filter.OpE, Value: "4.2"}
	args := `a = 4.2`
	cmd := strings.Split(args," ")

	//dbMock.GetUserNameMock.Expect(42).Return("Bob", nil)
	//dbMock.GetOrderItemsMock.Expect(100500).Return([]uint64{100, 200, 250}, nil)

	line:="a,b,c"

	fileCols := strings.Split(line, ",")
	// process get columns name from cmd
	flgCols := strings.Split(line, ",")



	colsIdx := map[string]int{
		"a":0,
		"b":0,
		"c":0,
	}

	colsMask := map[string]int{
		"a": 1,
		"b": 1,
		"c": 1,
	}

	ParseMock.ParseConditionMock.Expect(cmd,colsMask).Return(cond, nil)
	ParseMock.ParseHeadingMock.Expect(fileCols,flgCols).Return(colsMask,colsIdx)

	//собственно пробуем тестить функцию калькулятера (в которой эти два метода дают вот эти данные)
	filter, err := filter.NewFilter(args,fileCols,flgCols)
	//res, err := calc.ProcessOrder(42, 100500)
	require.NoError(t, err)
	fmt.Println("%w",filter)
	//assert.Equal(t, "user Bob spent $550", filter)
}