// 创建测试用户和数据库
db = db.getSiblingDB('test');

db.createUser({
  user: 'test',
  pwd: '',
  roles: [
    {
      role: 'readWrite',
      db: 'test'
    }
  ]
});

// 创建一些初始集合
db.createCollection('users');
db.createCollection('games');
db.createCollection('rooms');

print('MongoDB initialized successfully');
