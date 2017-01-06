// This is a small bot that messages someone (ZX9TZZ7P) and replies to everything with a qouted echo
package main

import (
	"fmt"
	"log"
	"os"
	"time"
	"strings"

	"github.com/o3ma/o3"
)

func main() {

	var (
		pass    = []byte{0xA, 0xB, 0xC, 0xD, 0xE}
		tr      o3.ThreemaRest
		idpath  = "threema.id"
		abpath  = "address.book"
		gdpath  = "group.directory"
		tid     o3.ThreemaID
		pubnick = "parrot"
		rid     = "8S3HMY9Z"
		err     error
	)

	// check whether an id file exists or else create a new one
	if _, err := os.Stat(idpath); err != nil {

		fmt.Printf("Creating new identity\n")
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

	tid.Nick = o3.NewPubNick(pubnick)
	fmt.Printf("My ID: %s [%s]\n", tid.Nick, tid.String())
	//fmt.Printf("My ID: %#v\n", tid)

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

	// check if we know the remote ID for
	// (just demonstration purposes \bc sending and receiving functions do this lookup for us)
	if _, b := ctx.ID.Contacts.Get(rid); b == false {
		//retrieve the ID from Threema's servers
		remoteID := o3.NewIDString(rid)
		fmt.Printf("Retrieving %s from directory server\n", remoteID.String())
		remoteContact, err := tr.GetContactByID(remoteID)
		if err != nil {
			log.Fatal(err)
		}
		// add them to our address book
		ctx.ID.Contacts.Add(remoteContact)

		//and save the address book
		fmt.Printf("Saving addressbook to %s\n", abpath)
		err = ctx.ID.Contacts.SaveTo(abpath)
		if err != nil {
			fmt.Println("saving addressbook failed")
			log.Fatal(err)
		}
	}

	remoteID, _ := ctx.ID.Contacts.Get(rid)
	fmt.Printf("Remote ID: %s [%s]\n", remoteID.Name, rid)
	//fmt.Printf("Remote ID: %#v\n", remoteID)

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

// -----------
// MEDIA

		case o3.ImageMessage:
			// TODO: display the image if you like
			fmt.Printf("ImageMessage: [%s] %s.\n%v\n", msg.Sender().String(), "", msg)

		case o3.AudioMessage:
			// TODO: play the audio if you like
			fmt.Printf("AudioMessage: [%s] %s.\n%v\n", msg.Sender().String(), "", msg)

// -----------
// TEXT
		case o3.TextMessage:

			fmt.Printf("TextMessage: ~%s [%s]: %s\n", msg.PubNick(), msg.Sender().String(), msg.Text())

			// continue only if it's not a message we sent to ourselves, avoid recursive qoutes of qoutes
			if (tid.String() == msg.Sender().String()) {
				continue
			}

			// ---------------------------------
			// Check senders name and compare it to address-book
			// Prefere addressbook name, but if missing, use ~PubNick 
			// TODO : put addressbook stuff into library
			var pubNick, sender string

			pubNick = fmt.Sprintf("~%s", msg.PubNick().String()) // Threema uses tilde-prefix for sender assined names
			sender = msg.Sender().String()
			remoteID := o3.NewIDString(sender)
			
			// addressbook contact missing, so add it
			if _, b := ctx.ID.Contacts.Get(sender); b == false {
				fmt.Printf("contact missing in addressbook, so add it\n")

				//retrieve the ID from Threema's servers
				fmt.Printf("Retrieving %s from directory server\n", remoteID.String())
				remoteContact, err := tr.GetContactByID(remoteID)
				if err != nil {
					log.Fatal(err)
				}
				// add pubNix the sender told us
				remoteContact.Name = pubNick 
				// add them to our address book
				ctx.ID.Contacts.Add(remoteContact)
				// save the address book
				fmt.Printf("Saving addressbook to %s\n", abpath)
				err = ctx.ID.Contacts.SaveTo(abpath)
				if err != nil {
					fmt.Println("saving addressbook failed")
					log.Fatal(err)
				}
			}
			
			// addressbook use contact from
			remoteContact, _ := ctx.ID.Contacts.Get(sender)

			// addressbook does not contain name, so set senders information
			if (remoteContact.Name == "" ) {
				fmt.Printf("addressbook does not contain name, so set senders information: %s\n", pubNick)
				// add pubNic the sender told us
				remoteContact.Name = pubNick 
				// add them to our address book
				ctx.ID.Contacts.Add(remoteContact)
				// save the address book
				fmt.Printf("Saving addressbook to %s\n", abpath)
				err = ctx.ID.Contacts.SaveTo(abpath)
				if err != nil {
					fmt.Println("saving addressbook failed")
					log.Fatal(err)
				}
			}

			// addressbook contains senders tilde-name and is updatet by sender, so change it
			if (remoteContact.Name[0] == '~' && remoteContact.Name != pubNick ) {
				fmt.Printf("addressbook contains senders tilde-name and name is updatet, so change it from %s to %s\n", remoteContact.Name, pubNick)
				// set the new pubNic the sender told us
				remoteContact.Name = pubNick 
				// add them to our address book
				ctx.ID.Contacts.Add(remoteContact)
				// save the address book
				fmt.Printf("Saving addressbook to %s\n", abpath)
				err = ctx.ID.Contacts.SaveTo(abpath)
				if err != nil {
					fmt.Println("saving addressbook failed")
					log.Fatal(err)
				}
			}

			// output the message sent
			fmt.Printf("TextMessage: %s (%s) [%s]: %s\n", remoteContact.Name, pubNick, msg.Sender().String(), msg.Text())

			// confirm to the sender that we received the message
			// this is how one can send messages manually without helper functions like "SendTextMessage"
			drm, err := o3.NewDeliveryReceiptMessage(&ctx, msg.Sender().String(), msg.ID(), o3.MSGDELIVERED)
			if err != nil {
				log.Fatal(err)
			}
			sendMsgChan <- drm

			// confirm to the sender that we read the message
			red, err := o3.NewDeliveryReceiptMessage(&ctx, msg.Sender().String(), msg.ID(), o3.MSGREAD)
			if err != nil {
				log.Fatal(err)
			}
			sendMsgChan <- red

			// give a thumbs up
			//upm, err := o3.NewDeliveryReceiptMessage(&ctx, msg.Sender().String(), msg.ID(), o3.MSGAPPROVED)
			//if err != nil {
			//	log.Fatal(err)
			//}
			//sendMsgChan <- upm

			// respond with a quote of what was send to us.
			qoute := fmt.Sprintf("> %s: %s\n%s", msg.Sender(), strings.Replace(msg.Text(), "\n", "\n> ", -1), "Exactly!")
			// we use the convinient "SendTextMessage" function to send
			err = ctx.SendTextMessage(msg.Sender().String(), qoute, sendMsgChan)
			if err != nil {
				log.Fatal(err)
			}

// -----------
// GROUP
		case o3.GroupTextMessage:
			// TODO: ERROR: SendGroupTextMessage does not send to group
			fmt.Printf("GroupTextMessage: ~%s [%s] for Group [%x] created by [%s]\n", msg.PubNick(), msg.Sender(), msg.GroupID(), msg.GroupCreator())
			fmt.Printf("GroupTextMessage: Text: %s\n", msg.Text())

			group, ok := ctx.ID.Groups.Get(msg.GroupCreator(), msg.GroupID())
			if ok {
				time.Sleep(500 * time.Millisecond)
				qoute := fmt.Sprintf("> %s: %s\n%s", msg.Sender(), msg.Text(), "Exactly in group!")
				ctx.SendGroupTextMessage(group, qoute, sendMsgChan)
			} else {
				fmt.Printf("ERROR sending to group [%x] by [%s].\n", msg.GroupID(), msg.GroupCreator())
			}

		case o3.GroupManageSetNameMessage:
			// TODO: create group-list-entry if missing, update internal group-name
			fmt.Printf("Group [%x] is now called %s\n", msg.GroupID(), msg.Name())

		case o3.GroupManageSetMembersMessage:
			// TODO: this should be done in Add()
			_, ok := ctx.ID.Groups.Get(msg.Sender(), msg.GroupID())
			members := msg.Members()
			if !ok {
				// replace our id with group creator id
				// \bc we know we are in the group, but we don't know who the creator is
				for i := range members {
					if members[i] == ctx.ID.ID {
						members[i] = msg.Sender()
					}
				}
			}

			// TODO: add only adds if the group is new so updates on the member list do not work yet
			ctx.ID.Groups.Add(o3.Group{CreatorID: msg.Sender(), GroupID: msg.GroupID(), Members: members})
			ctx.ID.Groups.SaveToFile(gdpath)
			fmt.Printf("Group [%x] now includes %v\n", msg.GroupID(), msg.Members())

		case o3.GroupMemberLeftMessage:
			// TODO : change grp.Members?
			fmt.Printf("Member [%s] left the Group [%x]\n", msg.Sender(), msg.GroupID())

// -----------
// STATUS
		case o3.DeliveryReceiptMessage:
			switch msg.Status() {
			case o3.MSGDELIVERED:
				fmt.Printf("Message [%x] was received by ~%s [%s]\n", msg.MsgID(), msg.PubNick(), msg.Sender())
			case o3.MSGREAD:
				fmt.Printf("Message [%x] was read by ~%s [%s]\n", msg.MsgID(), msg.PubNick(), msg.Sender())
			case o3.MSGAPPROVED:
				fmt.Printf("Message [%x] was approved (thumb up) by ~%s [%s]\n", msg.MsgID(), msg.PubNick(), msg.Sender())
			case o3.MSGDISAPPROVED:
				fmt.Printf("Message [%x] was disapproved (thumb down) by ~%s [%s]\n", msg.MsgID(), msg.PubNick(), msg.Sender())
			default:
				fmt.Printf("Unknown Status. Message [%x] has been acknowledged by ~%s [%s] BUT Status unknown: %s\n", msg.MsgID(), msg.MsgID(), msg.PubNick(), msg.Sender(),  msg.Status())
				fmt.Printf("Message: %#v\n", msg)
			}

		case o3.TypingNotificationMessage:
			switch msg.OnOff {
				case 0:
					fmt.Printf("Contact ~%s [%s] is not typing.\n", msg.PubNick(), msg.Sender())
				case 1:
					fmt.Printf("Contact ~%s [%s] is typing.\n", msg.PubNick(), msg.Sender())
				default:
					fmt.Printf("Unknown typing notification for %s: %s \n", msg.Sender(), msg.PubNick(), msg.OnOff)
			}

		default:
			fmt.Printf("Unknown message type from: %s\nContent: %#v", msg.Sender(), msg)

		}
	}

}
