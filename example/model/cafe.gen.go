// Code generated by es2go. DO NOT EDIT.

package model

import "time"

// CafeDocJson 咖啡表
type CafeDocJson struct {
	Address       string           `json:"address" es:"type:text"`
	AverageRating float64          `json:"average_rating" es:"type:float"`
	CafeName      string           `json:"cafe_name" es:"type:text;keyword"` // 名称
	DateAdded     time.Time        `json:"date_added" es:"type:date"`
	Location      []float64        `json:"location" es:"type:geo_point"` // 经纬度
	MenuItems     *MenuItemsNested `json:"menu_items" es:"type:nested"`  // 菜单列表
	PhoneNumber   string           `json:"phone_number" es:"type:keyword"`
	ReviewCount   int64            `json:"review_count" es:"type:integer"`
	Website       string           `json:"website" es:"type:keyword"`
}

// MenuItemsNested 菜单列表
type MenuItemsNested struct {
	Category string       `json:"category" es:"type:keyword"`
	Items    *ItemsNested `json:"items" es:"type:nested"`
}

// ItemsNested .
type ItemsNested struct {
	Available   bool    `json:"available" es:"type:boolean"`
	Ingredients string  `json:"ingredients" es:"type:text;keyword"`
	ItemName    string  `json:"item_name" es:"type:text"`
	Price       float64 `json:"price" es:"type:float"` // 价格
	Size        string  `json:"size" es:"type:keyword"`
}
