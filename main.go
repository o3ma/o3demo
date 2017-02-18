// This is a small bot that messages someone (ZX9TZZ7P) and replies to everything with a qouted echo
package main

import (
	"log"
	"fmt"
	"os"

	"github.com/o3ma/o3"
)


func main() {
	var (
		pass    = []byte{0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37} // 01234567
		idpath  = "threema.id"
		abpath  = "address.book"
		gdpath  = "group.directory"
		pubnick = "parrot"
		rid     = "ZX9TZZ7P" // e.g. ZX9TZZ7P
		testMsg = "Say something!"
	)

	tr, tid, ctx, receiveMsgChan, sendMsgChan := initialise(pass, idpath, abpath, gdpath, pubnick)

	go sendTestMsg(tr, abpath, rid, testMsg, ctx, sendMsgChan)

	receiveLoop(tid, gdpath, ctx, receiveMsgChan, sendMsgChan)
}


func initialise(pass []byte, idpath string, abpath string, gdpath string, pubnick string) (o3.ThreemaRest, o3.ThreemaID, o3.SessionContext, <-chan o3.ReceivedMsg, chan<- o3.Message) {
		var (
			tr      o3.ThreemaRest
			tid     o3.ThreemaID
		)

	// check whether an id file exists or else create a new one
	if _, err := os.Stat(idpath); err != nil {
		var err error
		tid, err = tr.CreateIdentity()
		if err != nil {
			fmt.Println("CreateIdentity failed")
			log.Fatal(err)
		}
		fmt.Printf("Saving ID to %s\n", idpath)
		err = tid.SaveToFile(idpath, pass)
		if err != nil {
			fmt.Println("saving ID failed")
			log.Fatal(err)
		}
	} else {
		fmt.Printf("Loading ID from %s\n", idpath)
		tid, err = o3.LoadIDFromFile(idpath, pass)
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("Using ID: %s\n", tid.String())

	tid.Nick = o3.NewPubNick(pubnick)

	ctx := o3.NewSessionContext(tid)

	//check if we can load an addressbook
	if _, err := os.Stat(abpath); !os.IsNotExist(err) {
		fmt.Printf("Loading addressbook from %s\n", abpath)
		err = ctx.ID.Contacts.LoadFromFile(abpath)
		if err != nil {
			fmt.Println("loading addressbook failed")
			log.Fatal(err)
		}
	}

	//check if we can load a group directory
	if _, err := os.Stat(gdpath); !os.IsNotExist(err) {
		fmt.Printf("Loading group directory from %s\n", gdpath)
		err = ctx.ID.Groups.LoadFromFile(gdpath)
		if err != nil {
			fmt.Println("loading group directory failed")
			log.Fatal(err)
		}
	}

	// let the session begin
	fmt.Println("Starting session")
	sendMsgChan, receiveMsgChan, err := ctx.Run()
	if err != nil {
		log.Fatal(err)
	}

	return tr, tid, ctx, receiveMsgChan, sendMsgChan
}


func sendTestMsg(tr o3.ThreemaRest, abpath string, rid string, testMsg string, ctx o3.SessionContext, sendMsgChan chan<- o3.Message) {
	// check if we know the remote ID for
	// (just demonstration purposes \bc sending and receiving functions do this lookup for us)
	if _, b := ctx.ID.Contacts.Get(rid); b == false {
		//retrieve the ID from Threema's servers
		myID := o3.NewIDString(rid)
		fmt.Printf("Retrieving %s from directory server\n", myID.String())
		myContact, err := tr.GetContactByID(myID)
		if err != nil {
			log.Fatal(err)
		}
		// add them to our address book
		ctx.ID.Contacts.Add(myContact)

		//and save the address book
		fmt.Printf("Saving addressbook to %s\n", abpath)
		err = ctx.ID.Contacts.SaveTo(abpath)
		if err != nil {
			fmt.Println("saving addressbook failed")
			log.Fatal(err)
		}
	}

	// send our initial message to our recipient
	err, tm := ctx.SendTextMessage(rid, testMsg, sendMsgChan)
	fmt.Println("Sending initial message [" + fmt.Sprintf("%x", tm.ID()) + "] to " + rid + ": " + testMsg + "\n--------------------\n")
	if err != nil {
		log.Fatal(err)
	}
}


func receiveLoop(tid o3.ThreemaID, gdpath string, ctx o3.SessionContext, receiveMsgChan <-chan o3.ReceivedMsg, sendMsgChan chan<- o3.Message) {

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
			// respond with a quote of what was send to us.
			fmt.Printf("\nMessage from %s: %s\n--------------------\n\n", msg.Sender(), msg.Text())
			
			// but only if it's no a message we sent to ourselves, avoid recursive neverending qoutes
			if (tid.String() == msg.Sender().String()) {
				continue
			}
			
			// to make the quote render nicely in the app we use "markdown"
			// of the form "> PERSONWEQUOTE: Text of qoute\nSomething we wanna add."
			qoute := fmt.Sprintf("> %s: %s\n%s", msg.Sender(), msg.Text(), "Exactly!")
			// we use the convinient "SendTextMessage" function to send
			err, _ := ctx.SendTextMessage(msg.Sender().String(), qoute, sendMsgChan)
			if err != nil {
				log.Fatal(err)
			}
			// confirm to the sender that we received the message
			// this is how one can send messages manually without helper functions like "SendTextMessage"
			drm, err := o3.NewDeliveryReceiptMessage(&ctx, msg.Sender().String(), msg.ID(), o3.MSGDELIVERED)
			if err != nil {
				log.Fatal(err)
			}
			sendMsgChan <- drm
			// give a thumbs up
			upm, err := o3.NewDeliveryReceiptMessage(&ctx, msg.Sender().String(), msg.ID(), o3.MSGAPPROVED)
			if err != nil {
				log.Fatal(err)
			}
			sendMsgChan <- upm
		case o3.GroupTextMessage:
			fmt.Printf("%s for Group [%x] created by [%s]:\n%s\n", msg.Sender(), msg.GroupID(), msg.GroupCreator(), msg.Text())
			group, ok := ctx.ID.Groups.Get(msg.GroupCreator(), msg.GroupID())
			if ok {
				ctx.SendGroupTextMessage(group, "This is a group reply!", sendMsgChan)
			}
		case o3.GroupManageSetNameMessage:
			fmt.Printf("Group [%x] is now called %s\n", msg.GroupID(), msg.Name())
			ctx.ID.Groups.Upsert(o3.Group{CreatorID: msg.Sender(), GroupID: msg.GroupID(), Name: msg.Name()})
			ctx.ID.Groups.SaveToFile(gdpath)
		case o3.GroupManageSetMembersMessage:
			fmt.Printf("Group [%x] now includes %v\n", msg.GroupID(), msg.Members())
			ctx.ID.Groups.Upsert(o3.Group{CreatorID: msg.Sender(), GroupID: msg.GroupID(), Members: msg.Members()})
			ctx.ID.Groups.SaveToFile(gdpath)
		case o3.GroupMemberLeftMessage:
			fmt.Printf("Member [%s] left the Group [%x]\n", msg.Sender(), msg.GroupID())
		case o3.DeliveryReceiptMessage:
			fmt.Printf("Message [%x] has been acknowledged by the server.\n", msg.MsgID())
		case o3.TypingNotificationMessage:
			fmt.Printf("Typing Notification from %s: [%x]\n", msg.Sender(), msg.OnOff)
		default:
			fmt.Printf("Unknown message type from: %s\nContent: %#v", msg.Sender(), msg)
		}
	}
}
