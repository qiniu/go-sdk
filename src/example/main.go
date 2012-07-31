package main

func main() {
	qboxServer := NewQbox()
	qboxServer.CheckQboxUser("test")

	table := "test"
	key := "key"
	//Put file
	ret := qboxServer.Put("test", table, key, "./tmp/test")
	if ret == true {
		//Publish table
		ret := qboxServer.Publish("test", "http://www.domain.com", table)
		if ret == true {
			//You can access http://www.domain.com/key
		}
	}

	//When you want to delete key of table
	if true {
		ret := qboxServer.Delete("test", table, key)
		if ret == true {
			// You access http://www.domain.com/key fail
		}
	}

	//When you want to drop table
	if true {
		ret := qboxServer.Drop("test", table)
		if ret == true {
			// it drop table
			// You access http://www.domain.com/key fail
		}
	}
}
