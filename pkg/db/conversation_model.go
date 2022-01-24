package db

import (
	"errors"
	"gorm.io/gorm"
	"open_im_sdk/pkg/utils"
	"time"
)

func (d *DataBase) GetAllConversationList() ([]*LocalConversation, error) {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	var conversationList []LocalConversation
	err := utils.Wrap(d.conn.Where("latest_msg_send_time != ?", time.Time{}).Order("case when is_pinned=1 then 0 else 1 end,max(latest_msg_send_time,draft_text_time) DESC").Find(&conversationList).Error,
		"GetFriendList failed")
	var transfer []*LocalConversation
	for _, v := range conversationList {
		v1 := v
		transfer = append(transfer, &v1)
	}
	return transfer, err
}
func (d *DataBase) GetConversationListSplit(offset, count int) ([]*LocalConversation, error) {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	var conversationList []LocalConversation
	err := utils.Wrap(d.conn.Where("latest_msg_send_time != ?", time.Time{}).Order("case when is_pinned=1 then 0 else 1 end,max(latest_msg_send_time,draft_text_time) DESC").Offset(offset).Limit(count).Find(&conversationList).Error,
		"GetFriendList failed")
	var transfer []*LocalConversation
	for _, v := range conversationList {
		v1 := v
		transfer = append(transfer, &v1)
	}
	return transfer, err
}
func (d *DataBase) BatchInsertConversationList(conversationList []*LocalConversation) error {
	if conversationList == nil {
		return nil
	}
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	return utils.Wrap(d.conn.Create(conversationList).Error, "BatchInsertConversationList failed")
}
func (d *DataBase) InsertConversation(conversationList *LocalConversation) error {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	return utils.Wrap(d.conn.Create(conversationList).Error, "InsertConversation failed")
}
func (d *DataBase) GetConversation(conversationID string) (*LocalConversation, error) {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	var c LocalConversation
	return &c, utils.Wrap(d.conn.Where("conversation_id = ?",
		conversationID).Find(&c).Error, "GetConversation failed")
}

func (d *DataBase) UpdateConversation(c *LocalConversation) error {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	t := d.conn.Updates(c)
	if t.RowsAffected == 0 {
		return utils.Wrap(errors.New("RowsAffected == 0"), "no update")
	}
	return utils.Wrap(t.Error, "UpdateConversation failed")
}
func (d *DataBase) BatchUpdateConversationList(conversationList []*LocalConversation) error {
	for _, v := range conversationList {
		err := d.UpdateConversation(v)
		if err != nil {
			return utils.Wrap(err, "BatchUpdateConversationList failed")
		}

	}
	return nil
}
func (d *DataBase) ConversationIfExists(conversationID string) (bool, error) {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	var count int64
	t := d.conn.Model(&LocalConversation{}).Where("conversation_id = ?",
		conversationID).Count(&count)
	if t.Error != nil {
		return false, utils.Wrap(t.Error, "ConversationIfExists get failed")
	}
	if count != 1 {
		return false, nil
	} else {
		return true, nil
	}
}

//Reset the conversation is equivalent to deleting the conversation,
//and the GetAllConversation or GetConversationListSplit interface will no longer be obtained.
func (d *DataBase) ResetConversation(conversationID string) error {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	c := LocalConversation{ConversationID: conversationID, UnreadCount: 0, LatestMsg: "", LatestMsgSendTime: 0, DraftText: "", DraftTextTime: 0}
	t := d.conn.Select("unread_count", "latest_msg", "latest_msg_send_time", "draft_text", "draft_text_time").Updates(c)
	if t.RowsAffected == 0 {
		return utils.Wrap(errors.New("RowsAffected == 0"), "no update")
	}
	return utils.Wrap(t.Error, "ResetConversation failed")
}

//Clear the conversation, which is used to delete the conversation history message and clear the conversation at the same time.
//The GetAllConversation or GetConversationListSplit interface can still be obtained,
//but there is no latest message.
func (d *DataBase) ClearConversation(conversationID string) error {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	c := LocalConversation{ConversationID: conversationID, UnreadCount: 0, LatestMsg: "", DraftText: "", DraftTextTime: 0}
	t := d.conn.Select("unread_count", "latest_msg", "draft_text", "draft_text_time").Updates(c)
	if t.RowsAffected == 0 {
		return utils.Wrap(errors.New("RowsAffected == 0"), "no update")
	}
	return utils.Wrap(t.Error, "ClearConversation failed")
}
func (d *DataBase) SetConversationDraft(conversationID, draftText string) error {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	t := d.conn.Exec("update conversation set draft_text=?,draft_text_time=?,latest_msg_send_time=case when latest_msg_send_time=? then ? else latest_msg_send_time  end where conversation_id=?",
		draftText, time.Now(), time.Time{}, time.Now(), conversationID)
	if t.RowsAffected == 0 {
		return utils.Wrap(errors.New("RowsAffected == 0"), "no update")
	}
	return utils.Wrap(t.Error, "SetConversationDraft failed")
}
func (d *DataBase) RemoveConversationDraft(conversationID, draftText string) error {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	c := LocalConversation{ConversationID: conversationID, DraftText: draftText, DraftTextTime: 0}
	t := d.conn.Select("draft_text", "draft_text_time").Updates(c)
	if t.RowsAffected == 0 {
		return utils.Wrap(errors.New("RowsAffected == 0"), "no update")
	}
	return utils.Wrap(t.Error, "RemoveConversationDraft failed")
}
func (d *DataBase) UnPinConversation(conversationID string, isPinned int) error {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	t := d.conn.Exec("update conversation set is_pinned=?,draft_text_time=case when draft_text=? then ? else draft_text_time  end where conversation_id=?",
		isPinned, "", time.Time{}, conversationID)
	if t.RowsAffected == 0 {
		return utils.Wrap(errors.New("RowsAffected == 0"), "no update")
	}
	return utils.Wrap(t.Error, "UnPinConversation failed")
}

func (d *DataBase) UpdateColumnsConversation(conversationID string, args map[string]interface{}) error {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	c := LocalConversation{ConversationID: conversationID}
	t := d.conn.Model(&c).Updates(args)
	if t.RowsAffected == 0 {
		return utils.Wrap(errors.New("RowsAffected == 0"), "no update")
	}
	return utils.Wrap(t.Error, "UpdateColumnsConversation failed")
}
func (d *DataBase) IncrConversationUnreadCount(conversationID string) error {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	c := LocalConversation{ConversationID: conversationID}
	t := d.conn.Model(&c).Update("unread_count", gorm.Expr("unread_count+?", 1))
	if t.RowsAffected == 0 {
		return utils.Wrap(errors.New("RowsAffected == 0"), "no update")
	}
	return utils.Wrap(t.Error, "IncrConversationUnreadCount failed")
}
func (d *DataBase) GetTotalUnreadMsgCount() (totalUnreadCount int64, err error) {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	var result []int64
	err = d.conn.Model(&LocalConversation{}).Pluck("unread_count", &result).Error
	if err != nil {
		return totalUnreadCount, utils.Wrap(errors.New("GetTotalUnreadMsgCount err"), "GetTotalUnreadMsgCount err")
	}
	for _, v := range result {
		totalUnreadCount += v
	}
	return totalUnreadCount, nil
}

func (d *DataBase) SetMultipleConversationRecvMsgOpt(conversationIDList []string, opt int) (err error) {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	t := d.conn.Model(&LocalConversation{}).Where("conversation_id IN ?", conversationIDList).Updates(map[string]interface{}{"recv_msg_opt": opt})
	if t.RowsAffected == 0 {
		return utils.Wrap(errors.New("RowsAffected == 0"), "no update")
	}
	return utils.Wrap(t.Error, "SetMultipleConversationRecvMsgOpt failed")
}

func (d *DataBase) GetMultipleConversation(conversationIDList []string) (result []*LocalConversation, err error) {
	d.mRWMutex.Lock()
	defer d.mRWMutex.Unlock()
	var conversationList []LocalConversation
	err = utils.Wrap(d.conn.Where("conversation_id IN ?", conversationIDList).Find(&conversationList).Error, "GetMultipleConversation failed")
	for _, v := range conversationList {
		v1 := v
		result = append(result, &v1)
	}
	return result, err
}