type SwipePoint = {
  x: number;
  y: number;
};

export function getSwipeCategoryOffset(start: SwipePoint, end: SwipePoint) {
  const deltaX = end.x - start.x;
  const deltaY = end.y - start.y;
  if (Math.abs(deltaX) < 42 || Math.abs(deltaX) < Math.abs(deltaY) * 1.25) {
    return null;
  }
  return deltaX < 0 ? 1 : -1;
}
