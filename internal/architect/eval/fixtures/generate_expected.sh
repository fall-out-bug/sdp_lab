#!/bin/bash
# Script to generate expected.json files for golden test cases

FIXTURES_DIR="/Users/fall_out_bug/projects/vibe_coding/sdp_lab/.claude/worktrees/feature-F105/internal/architect/eval/fixtures"

# Go test cases
cat > "$FIXTURES_DIR/go-simple-cli/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "go",
      "all": ["go"]
    }
  ],
  "dependencies": [
    {
      "file": "go.mod",
      "language": "go",
      "dep_count": 5
    }
  ],
  "import_graph": {
    "nodes": 3,
    "edges": 4
  },
  "infra": {
    "containers": []
  }
}
EOF

cat > "$FIXTURES_DIR/go-multi-module/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "go",
      "all": ["go"]
    }
  ],
  "dependencies": [
    {
      "file": "go.mod",
      "language": "go",
      "dep_count": 10
    },
    {
      "file": "cmd/service/go.mod",
      "language": "go",
      "dep_count": 5
    },
    {
      "file": "pkg/api/go.mod",
      "language": "go",
      "dep_count": 3
    }
  ],
  "import_graph": {
    "nodes": 15,
    "edges": 25,
    "clusters": [
      {
        "id": "cmd",
        "packages": ["cmd/server"],
        "internal_edges": 3,
        "external_edges": 2
      },
      {
        "id": "pkg",
        "packages": ["pkg/api", "pkg/auth"],
        "internal_edges": 5,
        "external_edges": 3
      }
    ]
  },
  "infra": {
    "containers": [
      {
        "name": "api",
        "type": "service",
        "source": "Dockerfile"
      }
    ]
  }
}
EOF

cat > "$FIXTURES_DIR/go-grpc-service/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "go",
      "all": ["go", "protobuf"]
    }
  ],
  "dependencies": [
    {
      "file": "go.mod",
      "language": "go",
      "dep_count": 8
    }
  ],
  "import_graph": {
    "nodes": 12,
    "edges": 18
  },
  "infra": {
    "containers": [
      {
        "name": "grpc-server",
        "type": "service",
        "source": "Dockerfile"
      }
    ]
  },
  "specs": [
    {
      "kind": "protobuf",
      "path": "proto/service.proto"
    }
  ]
}
EOF

cat > "$FIXTURES_DIR/go-gin-api/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "go",
      "all": ["go"]
    }
  ],
  "dependencies": [
    {
      "file": "go.mod",
      "language": "go",
      "dep_count": 12,
      "signals": ["web_framework"]
    }
  ],
  "import_graph": {
    "nodes": 20,
    "edges": 35
  },
  "infra": {
    "containers": [
      {
        "name": "api",
        "type": "service",
        "source": "Dockerfile"
      },
      {
        "name": "postgres",
        "type": "database",
        "source": "docker-compose.yml"
      }
    ]
  }
}
EOF

# Python test cases
cat > "$FIXTURES_DIR/python-flask/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "python",
      "all": ["python"]
    }
  ],
  "dependencies": [
    {
      "file": "requirements.txt",
      "language": "python",
      "dep_count": 8
    }
  ],
  "import_graph": {
    "nodes": 10,
    "edges": 15
  },
  "infra": {
    "containers": [
      {
        "name": "web",
        "type": "service",
        "source": "Dockerfile"
      }
    ]
  }
}
EOF

cat > "$FIXTURES_DIR/python-fastapi/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "python",
      "all": ["python"]
    }
  ],
  "dependencies": [
    {
      "file": "requirements.txt",
      "language": "python",
      "dep_count": 10
    },
    {
      "file": "pyproject.toml",
      "language": "python",
      "dep_count": 5
    }
  ],
  "import_graph": {
    "nodes": 15,
    "edges": 20
  },
  "infra": {
    "containers": [
      {
        "name": "api",
        "type": "service",
        "source": "Dockerfile"
      }
    ]
  },
  "specs": [
    {
      "kind": "openapi",
      "path": "openapi.json"
    }
  ]
}
EOF

cat > "$FIXTURES_DIR/python-django/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "python",
      "all": ["python", "javascript"]
    }
  ],
  "dependencies": [
    {
      "file": "requirements.txt",
      "language": "python",
      "dep_count": 25
    }
  ],
  "import_graph": {
    "nodes": 40,
    "edges": 60
  },
  "infra": {
    "containers": [
      {
        "name": "web",
        "type": "service",
        "source": "Dockerfile"
      },
      {
        "name": "db",
        "type": "database",
        "source": "docker-compose.yml"
      },
      {
        "name": "cache",
        "type": "cache",
        "source": "docker-compose.yml"
      }
    ]
  }
}
EOF

# Java test cases
cat > "$FIXTURES_DIR/java-spring-boot/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "java",
      "all": ["java", "xml"]
    }
  ],
  "dependencies": [
    {
      "file": "pom.xml",
      "language": "java",
      "dep_count": 20
    }
  ],
  "import_graph": {
    "nodes": 30,
    "edges": 50
  },
  "infra": {
    "containers": [
      {
        "name": "app",
        "type": "service",
        "source": "Dockerfile"
      },
      {
        "name": "mysql",
        "type": "database",
        "source": "docker-compose.yml"
      }
    ]
  }
}
EOF

cat > "$FIXTURES_DIR/java-gradle-multi/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "java",
      "all": ["java", "kotlin"]
    }
  ],
  "dependencies": [
    {
      "file": "build.gradle",
      "language": "java",
      "dep_count": 15
    },
    {
      "file": "api/build.gradle",
      "language": "java",
      "dep_count": 8
    },
    {
      "file": "service/build.gradle",
      "language": "kotlin",
      "dep_count": 10
    }
  ],
  "import_graph": {
    "nodes": 25,
    "edges": 40,
    "clusters": [
      {
        "id": "api",
        "packages": ["com.example.api"],
        "internal_edges": 5,
        "external_edges": 3
      },
      {
        "id": "service",
        "packages": ["com.example.service"],
        "internal_edges": 8,
        "external_edges": 5
      }
    ]
  },
  "infra": {
    "containers": [
      {
        "name": "api",
        "type": "service",
        "source": "Dockerfile"
      },
      {
        "name": "service",
        "type": "service",
        "source": "Dockerfile"
      }
    ]
  }
}
EOF

# TypeScript/JavaScript test cases
cat > "$FIXTURES_DIR/typescript-nextjs/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "typescript",
      "all": ["typescript", "javascript", "css"]
    }
  ],
  "dependencies": [
    {
      "file": "package.json",
      "language": "typescript",
      "dep_count": 20
    }
  ],
  "import_graph": {
    "nodes": 50,
    "edges": 80
  },
  "infra": {
    "containers": [
      {
        "name": "web",
        "type": "service",
        "source": "Dockerfile"
      }
    ]
  }
}
EOF

cat > "$FIXTURES_DIR/typescript-nestjs/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "typescript",
      "all": ["typescript"]
    }
  ],
  "dependencies": [
    {
      "file": "package.json",
      "language": "typescript",
      "dep_count": 15
    }
  ],
  "import_graph": {
    "nodes": 20,
    "edges": 30
  },
  "infra": {
    "containers": [
      {
        "name": "api",
        "type": "service",
        "source": "Dockerfile"
      },
      {
        "name": "postgres",
        "type": "database",
        "source": "docker-compose.yml"
      }
    ]
  },
  "specs": [
    {
      "kind": "openapi",
      "path": "api/openapi.yaml"
    }
  ]
}
EOF

cat > "$FIXTURES_DIR/javascript-express/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "javascript",
      "all": ["javascript"]
    }
  ],
  "dependencies": [
    {
      "file": "package.json",
      "language": "javascript",
      "dep_count": 12
    }
  ],
  "import_graph": {
    "nodes": 15,
    "edges": 20
  },
  "infra": {
    "containers": [
      {
        "name": "api",
        "type": "service",
        "source": "Dockerfile"
      }
    ]
  }
}
EOF

cat > "$FIXTURES_DIR/javascript-monorepo/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "typescript",
      "all": ["typescript", "javascript"]
    }
  ],
  "dependencies": [
    {
      "file": "package.json",
      "language": "typescript",
      "dep_count": 10
    },
    {
      "file": "packages/api/package.json",
      "language": "typescript",
      "dep_count": 8
    },
    {
      "file": "packages/web/package.json",
      "language": "typescript",
      "dep_count": 15
    },
    {
      "file": "packages/mobile/package.json",
      "language": "typescript",
      "dep_count": 12
    }
  ],
  "import_graph": {
    "nodes": 60,
    "edges": 100,
    "clusters": [
      {
        "id": "packages/api",
        "packages": ["@monorepo/api"],
        "internal_edges": 10,
        "external_edges": 5
      },
      {
        "id": "packages/web",
        "packages": ["@monorepo/web"],
        "internal_edges": 15,
        "external_edges": 8
      },
      {
        "id": "packages/mobile",
        "packages": ["@monorepo/mobile"],
        "internal_edges": 12,
        "external_edges": 6
      }
    ]
  },
  "infra": {
    "containers": [
      {
        "name": "api",
        "type": "service",
        "source": "packages/api/Dockerfile"
      },
      {
        "name": "web",
        "type": "service",
        "source": "packages/web/Dockerfile"
      }
    ]
  }
}
EOF

# SQL test cases
cat > "$FIXTURES_DIR/sql-migration-dir/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "sql",
      "all": ["sql"]
    }
  ],
  "sql_analysis": {
    "tables": [
      {
        "name": "users",
        "schema": "public"
      },
      {
        "name": "posts",
        "schema": "public"
      },
      {
        "name": "comments",
        "schema": "public"
      },
      {
        "name": "migrations",
        "schema": "public"
      }
    ]
  }
}
EOF

cat > "$FIXTURES_DIR/sql-orm-gorm/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "go",
      "all": ["go"]
    }
  ],
  "sql_analysis": {
    "tables": [
      {
        "name": "User",
        "schema": "main"
      },
      {
        "name": "Profile",
        "schema": "main"
      },
      {
        "name": "Settings",
        "schema": "main"
      }
    ]
  },
  "generated": [
    {
      "path": "models/user.go",
      "reason": "gorm:model"
    },
    {
      "path": "models/profile.go",
      "reason": "gorm:model"
    }
  ]
}
EOF

cat > "$FIXTURES_DIR/sql-orm-sqlalchemy/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "python",
      "all": ["python"]
    }
  ],
  "sql_analysis": {
    "tables": [
      {
        "name": "users",
        "schema": "public"
      },
      {
        "name": "addresses",
        "schema": "public"
      },
      {
        "name": "orders",
        "schema": "public"
      }
    ]
  },
  "generated": [
    {
      "path": "models/user.py",
      "reason": "sqlalchemy:model"
    },
    {
      "path": "models/address.py",
      "reason": "sqlalchemy:model"
    }
  ]
}
EOF

cat > "$FIXTURES_DIR/sql-orm-prisma/expected.json" << 'EOF'
{
  "languages": [
    {
      "primary": "typescript",
      "all": ["typescript", "prisma"]
    }
  ],
  "sql_analysis": {
    "tables": [
      {
        "name": "User",
        "schema": "public"
      },
      {
        "name": "Post",
        "schema": "public"
      },
      {
        "name": "Comment",
        "schema": "public"
      }
    ]
  },
  "generated": [
    {
      "path": "node_modules/.prisma/client/index.ts",
      "reason": "prisma:client"
    }
  ]
}
EOF

echo "Generated expected.json files for all golden test cases"
