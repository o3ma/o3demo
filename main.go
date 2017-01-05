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
		pass    = []byte{0xA, 0xB, 0xC, 0xD, 0xE}
		tr      o3.ThreemaRest
		idpath  = "threema.id"
		abpath  = "address.book"
		tid     o3.ThreemaID
		pubnick = "parrot"
		rid     = "8S3HMY9Z"
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
		err = ctx.ID.Contacts.ImportFrom(abpath)
		if err != nil {
			fmt.Println("loading addressbook failed")
			log.Fatal(err)
		}
	}

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

	// let the session begin
	fmt.Println("Starting session")
	sendMsgChan, receiveMsgChan, err := ctx.Run()
	if err != nil {
		log.Fatal(err)
	}

	// send our initial message to our recipient
	fmt.Println("Sending initial message")
	err = ctx.SendTextMessage(rid, "Say something!", sendMsgChan)
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
			fmt.Printf("TextMessage: ~%s [%s]: %s\n", msg.PubNick(), msg.Sender().String(), msg.Text())

			// continue only if it's not a message we sent to ourselves, avoid recursive qoutes of qoutes
			if (tid.String() == msg.Sender().String()) {
				continue
			}
			
			// respond with a quote of what was send to us.
			// to make the quote render nicely in the app we use "markdown"
			// of the form "> PERSONWEQUOTE: Text of qoute\nSomething we wanna add."
			qoute := fmt.Sprintf("> %s: %s\n%s", msg.Sender(), msg.Text(), "Exactly!")
			// we use the convinient "SendTextMessage" function to send
			err = ctx.SendTextMessage(msg.Sender().String(), qoute, sendMsgChan)
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
			// TODO: ERROR: SendGroupTextMessage does not send to group
			fmt.Printf("GroupTextMessage: ~%s [%s] for Group [%x] created by [%s]\n", msg.PubNick(), msg.Sender(), msg.GroupID(), msg.GroupCreator())
			fmt.Printf("GroupTextMessage: Text: %s\n", msg.Text())

			// is there a list for GroupCreators groups? otherwise create on the fly
			if (ctx.ID.Groups == nil) || (ctx.ID.Groups[msg.GroupCreator()] == nil) {
				fmt.Printf("Group-list missing, have to create group-list.\n", msg.GroupID(), msg.GroupCreator())
				// create group-list if missing
				if ctx.ID.Groups == nil {
					ctx.ID.Groups = make( map[o3.IDString]map[[8]byte]o3.Group )
				}
				// create group-list-entry if missing
				_, ok := ctx.ID.Groups[msg.GroupCreator()]
				if !ok {
					fmt.Printf("Group [%x] is new by [%s]\n", msg.GroupID(), msg.GroupCreator())
					ctx.ID.Groups[msg.GroupCreator()] = make(map[[8]byte]o3.Group)
				}
			}

			// is this group known? otherwise create on the fly, not knowing all members :(
			// can you ask GroupCreator for Members? 
			group, ok := ctx.ID.Groups[msg.GroupCreator()][msg.GroupID()]
			if !ok {
				fmt.Printf("Group unknown, create group [%x] by [%s] on the fly, not knowing all members..\n", msg.GroupID(), msg.GroupCreator())
				var tmpMembers []o3.IDString
				tmpMembers = append(tmpMembers, msg.GroupCreator())
				tmpMembers = append(tmpMembers, msg.Sender())
				tmpMembers = append(tmpMembers, msg.Recipient())
				// what about duplicates?
				ctx.ID.Groups[msg.GroupCreator()][msg.GroupID()] = o3.Group{CreatorID: msg.GroupCreator(), GroupID: msg.GroupID(), Members: tmpMembers }
			}

			// send message to group, which should exist now
			group, ok = ctx.ID.Groups[msg.GroupCreator()][msg.GroupID()]
			if ok {
				ctx.SendGroupTextMessage(group, "This is a group reply!", sendMsgChan)
			} else {
				fmt.Printf("ERROR sending to group [%x] by [%s].\n", msg.GroupID(), msg.GroupCreator())
			}
			
		case o3.GroupManageSetNameMessage:
			// TODO: create group-list-entry if missing, update internal group-name
			fmt.Printf("Group [%x] is now called %s\n", msg.GroupID(), msg.Name())
			
		case o3.GroupManageSetMembersMessage:
			fmt.Printf("Group [%x] now includes %v\n", msg.GroupID(), msg.Members())

			// create group-list if missing
			if ctx.ID.Groups == nil {
				ctx.ID.Groups = make( map[o3.IDString]map[[8]byte]o3.Group )
			}

			// create group-list-entry if missing
			_, ok := ctx.ID.Groups[msg.Sender()]
			if !ok {
				fmt.Printf("Group [%x] is new by [%s]\n", msg.GroupID(), msg.Sender())
				ctx.ID.Groups[msg.Sender()] = make(map[[8]byte]o3.Group)
			}

			// assign members to group
			ctx.ID.Groups[msg.Sender()][msg.GroupID()] = o3.Group{CreatorID: msg.Sender(), GroupID: msg.GroupID(), Members: msg.Members()}

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
