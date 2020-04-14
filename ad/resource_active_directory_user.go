package ad

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"gopkg.in/ldap.v3"
)

func resourceUser() *schema.Resource {
	return &schema.Resource{
		Create: resourceADUserCreate,
		Read:   resourceADUserRead,
		Delete: resourceADUserDelete,

		Schema: map[string]*schema.Schema{
			"first_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"last_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"domain": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ou_distinguished_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"logon_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"password": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  nil,
				ForceNew: true,
			},
			"must_change_pw": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},
			"cannot_change_pw": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"password_not_expire": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
		},
	}
}

// function to add a user in AD:

func resourceADUserCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*ldap.Conn)
	firstName := d.Get("first_name").(string)
	lastName := d.Get("last_name").(string)
	domain := d.Get("domain").(string)
	pass := d.Get("password").(string)
	logonName := d.Get("logon_name").(string)
	must_change_pw := d.Get("must_change_pw").(bool)
	cannot_change_pw := d.Get("cannot_change_pw").(bool)
	password_not_expire := d.Get("password_not_expire").(bool)
	upn := logonName + "@" + domain
	desc := d.Get("description").(string)
	userName := firstName + " " + lastName
	var dnOfUser string // dnOfUser: distingished names uniquely identifies an entry to AD.
	dnOfUser += "CN=" + userName
	dnOfUser += "," + d.Get("ou_distinguished_name").(string)

	log.Printf("[DEBUG] dnOfUser: %s ", dnOfUser)
	log.Printf("[DEBUG] Adding user : %s ", userName)
	err := addUser(userName, logonName, dnOfUser, client, upn, lastName, pass, desc, must_change_pw, cannot_change_pw, password_not_expire)
	if err != nil {
		log.Printf("[ERROR] Error while adding user: %s ", err)
		return fmt.Errorf("Error while adding user %s", err)
	}
	log.Printf("[DEBUG] User Added success: %s", userName)
	//d.SetId(domain + "/" + userName)
	d.SetId(dnOfUser)
	return nil
}

// Function to read user in AD:

func resourceADUserRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*ldap.Conn)
	firstName := d.Get("first_name").(string)
	lastName := d.Get("last_name").(string)
	//domain := d.Get("domain").(string)
	userName := firstName + " " + lastName
	var dnOfUser string // dnOfUser: distingished names uniquely identifies an entry to AD.
	dnOfUser += d.Get("ou_distinguished_name").(string)

	log.Printf("[DEBUG] dnOfUser: %s ", dnOfUser)
	log.Printf("[DEBUG] Deleting user : %s ", userName)

	NewReq := ldap.NewSearchRequest(
		dnOfUser, // base dnOfUser.
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0,
		false,
		"(&(objectClass=User)(cn="+userName+"))", //applied filter
		[]string{"dnOfUser", "cn"},
		nil,
	)

	sr, err := client.Search(NewReq)
	if err != nil {
		log.Printf("[ERROR] while seaching user : %s", err)
		return fmt.Errorf("Error while searching  user : %s", err)
	}

	fmt.Println("[ERROR] Found " + strconv.Itoa(len(sr.Entries)) + " Entries")
	for _, entry := range sr.Entries {
		fmt.Printf("%s: %v\n", entry.DN, entry.GetAttributeValue("cn"))

	}

	if len(sr.Entries) == 0 {
		log.Println("[ERROR] user not found")
		d.SetId("")
	}
	return nil
}

//function to delete user from AD :

func resourceADUserDelete(d *schema.ResourceData, m interface{}) error {
	resourceADUserRead(d, m)
	if d.Id() == "" {
		log.Printf("[ERROR] user not found !!")
		return fmt.Errorf("[ERROR] Cannot find user")
	}
	client := m.(*ldap.Conn)
	firstName := d.Get("first_name").(string)
	lastName := d.Get("last_name").(string)
	//domain := d.Get("domain").(string)
	userName := firstName + " " + lastName
	var dnOfUser string
	dnOfUser += "CN=" + userName
	dnOfUser += "," + d.Get("ou_distinguished_name").(string)
	log.Printf("[DEBUG] dnOfUser: %s ", dnOfUser)
	log.Printf("[DEBUG] deleting user : %s ", userName)
	err := delUser(userName, dnOfUser, client)
	if err != nil {
		log.Printf("[ERROR] Error in deletion :%s", err)
		return fmt.Errorf("[ERROR] Error in deletion :%s", err)
	}
	log.Printf("[DEBUG] user Deleted success: %s", userName)
	return nil
}

// Helper function for adding user:
func addUser(userName string, logonName string, dnName string, adConn *ldap.Conn, upn string, lastName string, pass string, desc string,
	must_change_pw bool, cannot_change_pw bool, password_not_expire bool) error {
	userAccountControlValue := 544
	if cannot_change_pw {
		log.Printf("[DEBUG] addUser: setting cannot_change_pw true")
		userAccountControlValue += 64
	}
	if password_not_expire {
		log.Printf("[DEBUG] addUser: setting password_not_expire true")
		userAccountControlValue += 65536
	}
	log.Printf("[DEBUG] addUser: final userAccountControlValue computed: %s", strconv.Itoa(userAccountControlValue))
	a := ldap.NewAddRequest(dnName, nil) // returns a new AddRequest without attributes " with dn".
	a.Attribute("objectClass", []string{"organizationalPerson", "person", "top", "user"})
	a.Attribute("sAMAccountName", []string{logonName})
	a.Attribute("userPrincipalName", []string{upn})
	a.Attribute("name", []string{userName})
	a.Attribute("sn", []string{lastName})
	a.Attribute("userPassword", []string{pass})
	a.Attribute("userAccountControl", []string{strconv.Itoa(userAccountControlValue)}) //Default to enabled account
	if !must_change_pw {
		log.Printf("[DEBUG] addUser: setting must_change_pw false")
		a.Attribute("pwdLastSet", []string{"-1"})
	}

	if desc != "" {
		log.Printf("[DEBUG] addUser: setting description: %s", desc)
		a.Attribute("description", []string{desc})
	}

	log.Printf("[DEBUG] addUser: ldap request: %+v", a)

	err := adConn.Add(a)
	if err != nil {
		return err
	}
	return nil
}

//Helper function to delete user:

func delUser(userName string, dnName string, adConn *ldap.Conn) error {
	delReq := ldap.NewDelRequest(dnName, nil)
	err := adConn.Del(delReq)
	if err != nil {
		return err
	}
	return nil
}
