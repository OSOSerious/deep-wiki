package com.acme.ecommerce.user;

import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/users")
public class UserController {
    private final UserRepository repo;

    public UserController(UserRepository repo) { this.repo = repo; }

    @GetMapping
    public List<User> all() { return repo.findAll(); }

    @PostMapping
    public User create(@RequestBody User user) { return repo.save(user); }

    @GetMapping("/{id}")
    public User one(@PathVariable Long id) { return repo.findById(id).orElseThrow(); }
}