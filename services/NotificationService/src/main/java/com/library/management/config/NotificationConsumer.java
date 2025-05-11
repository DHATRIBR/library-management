package com.library.management.config;

import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Component;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.library.management.entity.Notification;
import com.library.management.service.NotificationService;

@Component
public class NotificationConsumer {

    @Autowired
    private NotificationService service;

    @KafkaListener(topics = {"book.due_date_reminder", "book.overdue", "book.available_reserved"}, groupId = "notification-group")
    public void listen(ConsumerRecord<String, String> record) {
        try {
            String value = record.value();
            ObjectMapper mapper = new ObjectMapper();
            JsonNode node = mapper.readTree(value);

            JsonNode userIdNode = node.get("userId");
            JsonNode messageNode = node.get("message");

            if (userIdNode == null || messageNode == null || userIdNode.isNull() || messageNode.isNull()) {
                System.err.println("Invalid message received: missing userId or message. Payload: " + value);
                return;
            }

            Notification notification = new Notification();
            notification.setUserId(userIdNode.asLong());
            notification.setMessage(messageNode.asText());

            service.saveNotification(notification);

        } catch (Exception e) {
            System.err.println("Failed to process message: " + record.value());
            e.printStackTrace();
        }
    }
}