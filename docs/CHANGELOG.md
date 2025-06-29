# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Complete documentation suite (README, API docs, Development guide)
- Comprehensive test coverage across all modules
- Docker support with optimized multi-stage builds
- Enhanced Makefile with all development commands

### Changed
- Improved daemon startup reliability and error handling
- Updated README with all current features and usage examples

### Fixed
- Daemon startup timeout issues resolved
- Protocol mismatch between client and server fixed
- Background daemon process stability improved

## [0.3.0] - 2024-01-XX

### Added
- **Daemon Mode**: Background daemon for asynchronous operations
  - `fman daemon start/stop/status/restart` commands
  - Unix domain socket communication
  - JSON-based message protocol
- **Queue System**: Job queue for background processing
  - `fman queue list/status/cancel/clear` commands
  - Job status tracking and progress monitoring
  - Automatic retry mechanism for failed jobs
- **Rules Engine**: Automated file organization rules
  - `fman rules list/add/remove/enable/disable/apply` commands
  - YAML-based rule configuration
  - Condition-based file matching
  - Multiple action types (move, copy, delete)
- **Duplicate File Detection**: Find and manage duplicate files
  - `fman duplicate` command with interactive mode
  - Hash-based duplicate detection
  - Configurable minimum file size threshold

### Enhanced
- **Improved Scanning**: Better performance and error handling
  - Parallel processing capabilities
  - Enhanced progress reporting
  - Better permission error handling
- **Advanced Search**: More search criteria and options
  - Size-based filtering
  - Date-based filtering
  - Path pattern matching
- **Configuration Management**: Enhanced config system
  - Better validation and error messages
  - Support for multiple AI providers

### Technical Improvements
- Comprehensive test suite with >70% coverage
- Mock interfaces for better testability
- Improved error handling and logging
- Better separation of concerns in architecture

## [0.2.0] - 2024-01-XX

### Added
- **AI Integration**: Support for Gemini and Ollama
  - `fman organize --ai` command
  - Configurable AI providers
  - Smart file organization suggestions
- **Advanced File Search**: Enhanced search capabilities
  - Pattern-based file name search
  - Metadata-based filtering
  - Fast indexed search
- **Configuration System**: YAML-based configuration
  - Auto-generated config file
  - AI provider settings
  - User preferences

### Enhanced
- **Database Schema**: Improved file metadata storage
  - File hash calculation for duplicate detection
  - Better indexing for faster searches
  - Metadata caching
- **Cross-Platform Support**: Better compatibility
  - Platform-specific system directory skipping
  - Improved permission handling
  - Path normalization

### Fixed
- Memory usage optimization during large scans
- Better error reporting for permission issues
- Improved handling of symbolic links

## [0.1.0] - 2024-01-XX

### Added
- **Initial Release**: Basic file management functionality
- **File Scanning**: Recursive directory scanning
  - `fman scan` command
  - SQLite database for metadata storage
  - File hash calculation
  - System directory auto-skip
- **Basic Search**: Simple file search
  - `fman find` command
  - Name-based pattern matching
- **Cross-Platform**: Support for macOS, Linux, Windows
- **Permission Handling**: Graceful handling of access restrictions
- **CLI Interface**: Cobra-based command structure

### Technical Features
- Go modules for dependency management
- SQLite3 for local data storage
- Viper for configuration management
- Cobra for CLI framework
- Comprehensive error handling

---

## Version History Summary

- **v0.3.0**: Daemon mode, queue system, rules engine, duplicate detection
- **v0.2.0**: AI integration, advanced search, configuration system  
- **v0.1.0**: Initial release with basic scanning and search

## Migration Guide

### From v0.2.x to v0.3.x

1. **Configuration**: No breaking changes to config format
2. **Database**: Automatic schema migration on first run
3. **Commands**: All existing commands remain compatible
4. **New Features**: New commands are opt-in and don't affect existing workflows

### From v0.1.x to v0.2.x

1. **Configuration File**: Config file format changed from JSON to YAML
   - Old config will be automatically migrated
   - Backup of old config created as `config.json.backup`

2. **Database Schema**: New fields added for AI integration
   - Automatic migration on first run
   - No data loss during migration

## Breaking Changes

### v0.3.0
- None

### v0.2.0
- Configuration file format changed from JSON to YAML
- Some internal API changes (affects only programmatic usage)

### v0.1.0
- Initial release (no breaking changes)

## Known Issues

### Current Version
- Large directory scans (>100k files) may consume significant memory
- AI suggestions quality depends on the configured model
- Windows symbolic link handling has limitations

### Resolved Issues
- ✅ Daemon startup timeout (fixed in v0.3.0)
- ✅ Permission errors on system directories (fixed in v0.1.1)
- ✅ Memory leaks during large scans (fixed in v0.2.1)

## Upcoming Features

### v0.4.0 (Planned)
- **Real-time File Watching**: Automatic index updates
- **Web Interface**: Browser-based file management
- **Plugin System**: Extensible architecture for custom actions
- **Cloud Integration**: Support for cloud storage providers
- **Performance Improvements**: Faster scanning and search

### v0.5.0 (Planned)
- **Distributed Mode**: Multi-machine file management
- **Advanced Analytics**: File usage patterns and insights
- **Machine Learning**: Improved AI suggestions based on user behavior
- **Mobile App**: Companion mobile application

## Contributing

We welcome contributions! Please see our [Development Guide](DEVELOPMENT.md) for details on:
- Setting up the development environment
- Running tests
- Submitting pull requests
- Code style guidelines

## Support

- **Documentation**: [README](../README.md) | [API Docs](API.md) | [Dev Guide](DEVELOPMENT.md)
- **Issues**: [GitHub Issues](https://github.com/devlikebear/fman/issues)
- **Discussions**: [GitHub Discussions](https://github.com/devlikebear/fman/discussions) 