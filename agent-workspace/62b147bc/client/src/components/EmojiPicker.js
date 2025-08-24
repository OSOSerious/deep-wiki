import React from 'react';
import Picker from 'emoji-picker-react';

function EmojiPicker({ onSelect }) {
  return <Picker onEmojiClick={onSelect} />;
}

export default EmojiPicker;