package com.library.management.entity;

import java.time.LocalDateTime;

import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;
import lombok.Data;

@Data
@Entity
@Table(name = "notifications")
public class Notification {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

//     @NotBlank(message = "userId is required")
    private Long userId;

    @NotBlank(message = "message is required")
    @Size(min = 5, max = 1000, message = "message must be between 5 and 1000 characters")
    private String message;

    private LocalDateTime timestamp = LocalDateTime.now();

    private boolean seen = false;

}
