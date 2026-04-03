---
name: object calisthenics enforcer
description: The Nine Rules of Object Calisthenics. Enforces small, focused, well-encapsulated objects through strict structural constraints. Acts as a guide toward SOLID, low complexity, and high cohesion.
applyTo: "**/*.{cs,go,ts,tsx,js,jsx,py}"
---

# The Nine Rules of Object Calisthenics

These rules are intentionally strict and act as a guide rather than absolute laws. Apply them to push toward better design; find the right balance for each context.

## The Rules

### 1. Only One Level of Indentation Per Method
Avoid nested loops and multiple `if` statements within a single method. Extract inner blocks into well-named methods to improve readability and enforce small, focused methods.

### 2. Don't Use the `else` Keyword
Use early returns or "fail fast" techniques to eliminate `else` blocks. Guard clauses and early exits reduce nesting and cognitive complexity.

### 3. Wrap All Primitives and Strings
Encapsulate primitive types (`int`, `float`, `string`, etc.) within dedicated classes. Prevents primitive obsession and enables domain-driven design with meaningful types (Value Objects).

### 4. First Class Collections
Any class that contains a collection should contain no other member variables. The collection and its related behaviors belong together; nothing else does.

### 5. One Dot Per Line
Enforce the Law of Demeter to reduce coupling. Avoid chaining method calls like `objectA.GetObjectB().DoSomething()`. Each object should talk only to its immediate neighbors.

### 6. Don't Abbreviate
Use descriptive, full names for classes, methods, and variables. If you feel the urge to abbreviate, the name is probably wrong or the method is doing too much.

### 7. Keep All Entities Small
- Classes: under 50–150 lines
- Packages/namespaces/modules: under 10 files

Small entities are easier to understand, test, and change.

### 8. No Classes With More Than Two Instance Variables
Limit class state to at most two instance variables. This constraint promotes high cohesion and drives decomposition into smaller, more specialized classes.

### 9. No Getters / Setters / Properties
Follow the "Tell, Don't Ask" principle. Objects should expose behavior, not data. Force objects to act on their own state rather than having external code read and manipulate it.

## Key Objectives

- **SOLID alignment:** Applying these rules naturally leads toward SRP and encapsulation.
- **Reduced complexity:** Discouraging nested logic makes code easier to understand and test.
- **Encapsulation:** Behaviors stay alongside the data they operate on (especially via Value Objects — Rule 3).
- **Improved maintainability:** Small, focused classes are easier to change and refactor.
