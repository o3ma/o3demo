// This is a small bot that messages someone (ZX9TZZ7P) and replies to everything with a qouted echo
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/o3ma/o3"
)

func main() {

	var (
		pass   = []byte{0xA, 0xB, 0xC, 0xD, 0xE}
		tr     o3.ThreemaRest
		idpath = "threema.id"
		//abpath = "address.book"
		tid o3.ThreemaID
	)

	// check whether an id file exists or else create a new one
	if _, err := os.Stat(idpath); os.IsNotExist(err) {

		tid, err := tr.CreateIdentity()
		if err != nil {
			fmt.Println("CreateIdentity failed")
			log.Fatal(err)
		}
		err = tid.SaveToFile(idpath, pass)
		if err != nil {
			fmt.Println("saving ID failed")
			log.Fatal(err)
		}
	} else {
		tid, err = o3.LoadIDFromFile(idpath, pass)
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("Use ID: %s\n", tid.String())

	ctx := o3.NewSessionContext(tid)

	// lookup our recipient on the threema directory server
	myID := o3.NewIDString("ZX9TZZ7P")
	myContact, err := tr.GetContactByID(myID)
	if err != nil {
		log.Fatal(err)
	}
	// add them to our address book
	ctx.ID.Contacts.Add(myContact)

	// let the session begin
	sendMsgChan, receiveMsgChan, err := ctx.Run()
	if err != nil {
		log.Fatal(err)
	}

	// send our initial message to our recipient
	err = ctx.SendTextMessage("ZX9TZZ7P", "Say something!", sendMsgChan)
	if err != nil {
		log.Fatal(err)
	}

	// handle incoming messages
	for receivedMessage := range receiveMsgChan {
		if receivedMessage.Err != nil {
			fmt.Printf("Error Receiving Message: %s\n", receivedMessage.Err)
			continue
		}
		switch msg := receivedMessage.Msg.(type) {
		case o3.ImageMessage:
			// display the image if you like
		case o3.AudioMessage:
			// play the audio if you like
		case o3.TextMessage:
			// echo a quoute
			qoute := fmt.Sprintf("> %s: %s\n%s", msg.Sender(), msg.Text(), "Exactly!")
			err = ctx.SendTextMessage(msg.Sender().String(), qoute, sendMsgChan)
			if err != nil {
				log.Fatal(err)
			}
		case o3.GroupTextMessage:
			fmt.Printf("%s for Group [%x] created by [%s]:\n%s\n", msg.Sender(), msg.GroupID(), msg.GroupCreator(), msg.Text())
		case o3.GroupManageSetNameMessage:
			fmt.Printf("Group [%x] is now called %s\n", msg.GroupID(), msg.Name())
		case o3.GroupManageSetMembersMessage:
			fmt.Printf("Group [%x] now includes %v\n", msg.GroupID(), msg.Members())
		case o3.GroupMemberLeftMessage:
			fmt.Printf("Member [%s] left the Group [%x]\n", msg.Sender(), msg.GroupID())
		case o3.DeliveryReceiptMessage:
			fmt.Printf("Message [%x] has been acknowledged by the server.\n", msg.MsgID)
		case o3.TypingNotificationMessage:
			fmt.Printf("Typing Notification from %s: [%x]\n", msg.Sender(), msg.OnOff)
		default:
			fmt.Printf("Unknown message type from: %s\nContent: %#v", msg.Sender(), msg)
		}
	}

}
