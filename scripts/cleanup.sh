#!/bin/bash

# 清理脚本 - 定期清理本地存储

set -e

echo "🧹 Anal Go Storage Cleanup Script"
echo "=================================="
echo ""

# 默认参数
DRY_RUN=${DRY_RUN:-true}
UPLOAD_EXPIRE=${UPLOAD_EXPIRE:-24}    # 上传文件保留24小时
DIAGRAM_EXPIRE=${DIAGRAM_EXPIRE:-7}   # diagram文件保留7天

# 解析命令行参数
while [[ $# -gt 0 ]]; do
  case $1 in
    --execute)
      DRY_RUN=false
      shift
      ;;
    --upload-expire)
      UPLOAD_EXPIRE="$2"
      shift 2
      ;;
    --diagram-expire)
      DIAGRAM_EXPIRE="$2"
      shift 2
      ;;
    --help)
      echo "Usage: $0 [options]"
      echo ""
      echo "Options:"
      echo "  --execute              Actually delete files (default: dry-run)"
      echo "  --upload-expire HOURS  Hours to keep upload files (default: 24)"
      echo "  --diagram-expire DAYS  Days to keep local diagrams (default: 7)"
      echo "  --help                 Show this help message"
      echo ""
      echo "Examples:"
      echo "  $0                     # Dry run with defaults"
      echo "  $0 --execute           # Actually delete files"
      echo "  $0 --upload-expire 12  # Keep uploads for 12 hours"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Use --help for usage information"
      exit 1
      ;;
  esac
done

# 显示配置
echo "Configuration:"
echo "  Dry Run: $DRY_RUN"
echo "  Upload Expire: ${UPLOAD_EXPIRE} hours"
echo "  Diagram Expire: ${DIAGRAM_EXPIRE} days"
echo ""

if [ "$DRY_RUN" = "true" ]; then
  echo "⚠️  Running in DRY RUN mode - no files will be deleted"
  echo "   Use --execute to actually delete files"
  echo ""
fi

# 在Docker容器中运行清理任务
docker exec anal_worker /app/cleanup \
  -dry-run=$DRY_RUN \
  -upload-expire=$UPLOAD_EXPIRE \
  -diagram-expire=$DIAGRAM_EXPIRE

echo ""
echo "✅ Cleanup script finished"
