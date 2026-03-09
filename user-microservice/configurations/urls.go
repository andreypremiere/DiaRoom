package configurations

// Пути для обращения к другим микросервисам

// Room-microservice
const BaseRoomUrl string = "http://room-microservice:81"

const GetRoomIdByUserId string = BaseRoomUrl + "/getRoomIdByUserId"